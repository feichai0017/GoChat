package k8s

import (
	"context"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bytedance/gopkg/util/logger"
	"github.com/feichai0017/GoChat/common/crpc/discov"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// Register ...
type Register struct {
	Options
	cli                 *kubernetes.Clientset
	serviceRegisterCh   chan *discov.Service
	serviceUnRegisterCh chan *discov.Service
	lock                sync.Mutex
	downServices        atomic.Value
	listeners           []func()
}

// NewK8sRegister ...
func NewK8sRegister(opts ...Option) (discov.Discovery, error) {
	opt := defaultOption
	for _, o := range opts {
		o(&opt)
	}

	r := &Register{
		Options:             opt,
		serviceRegisterCh:   make(chan *discov.Service),
		serviceUnRegisterCh: make(chan *discov.Service),
		lock:                sync.Mutex{},
		downServices:        atomic.Value{},
	}

	if err := r.init(context.TODO()); err != nil {
		return nil, err
	}

	return r, nil
}

// Init 初始化
func (r *Register) init(ctx context.Context) error {
	// 创建 in-cluster 配置
	config, err := rest.InClusterConfig()
	if err != nil {
		return err
	}

	// 创建 clientset
	r.cli, err = kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	go r.run()

	return nil
}

func (r *Register) run() {
	for {
		select {
		case service := <-r.serviceRegisterCh:
			logger.CtxInfof(context.TODO(), "K8s service registration is handled by K8s itself: %s", service.Name)
		case service := <-r.serviceUnRegisterCh:
			logger.CtxInfof(context.TODO(), "K8s service unregistration is handled by K8s itself: %s", service.Name)
		}
	}
}

func (r *Register) Name() string {
	return "k8s"
}

func (r *Register) AddListener(ctx context.Context, f func()) {
	r.listeners = append(r.listeners, f)
}

func (r *Register) NotifyListeners() {
	for _, listener := range r.listeners {
		listener()
	}
}

func (r *Register) Register(ctx context.Context, service *discov.Service) {
	// K8s中服务注册由K8s自身管理，这里只是为了兼容接口
	r.serviceRegisterCh <- service
}

func (r *Register) UnRegister(ctx context.Context, service *discov.Service) {
	// K8s中服务注销由K8s自身管理，这里只是为了兼容接口
	r.serviceUnRegisterCh <- service
}

func (r *Register) GetService(ctx context.Context, name string) *discov.Service {
	allServices := r.getDownServices()
	if val, ok := allServices[name]; ok {
		return val
	}

	// 防止并发获取service导致cache中的数据混乱
	r.lock.Lock()
	defer r.lock.Unlock()

	// 获取当前命名空间
	namespace := os.Getenv("POD_NAMESPACE")
	if namespace == "" {
		namespace = "default"
	}

	// 获取服务
	svc, err := r.cli.CoreV1().Services(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		logger.CtxErrorf(ctx, "get service from k8s error: %v", err)
		return &discov.Service{
			Name:      name,
			Endpoints: make([]*discov.Endpoint, 0),
		}
	}

	// 构建标签选择器
	labelSelector := labels.SelectorFromSet(svc.Spec.Selector)
	selectorStr := labelSelector.String()

	// 获取与服务关联的所有Pod
	pods, err := r.cli.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: selectorStr,
	})
	if err != nil {
		logger.CtxErrorf(ctx, "get pods from k8s error: %v", err)
		return &discov.Service{
			Name:      name,
			Endpoints: make([]*discov.Endpoint, 0),
		}
	}

	service := &discov.Service{
		Name:      name,
		Endpoints: make([]*discov.Endpoint, 0),
	}

	for _, pod := range pods.Items {
		// 只处理正在运行的Pod
		if pod.Status.Phase != "Running" {
			continue
		}

		// 获取Pod的IP地址
		podIP := pod.Status.PodIP
		if podIP == "" {
			continue
		}

		// 获取服务端口
		var port int
		if len(svc.Spec.Ports) > 0 {
			port = int(svc.Spec.Ports[0].Port)
		}

		endpoint := &discov.Endpoint{
			ServerName: pod.Name,
			IP:         podIP,
			Port:       port,
			Weight:     1,
			Enable:     true,
		}
		service.Endpoints = append(service.Endpoints, endpoint)
	}

	allServices[name] = service
	r.downServices.Store(allServices)

	// 启动Pod监控
	go r.watchPods(ctx, namespace, selectorStr, name, int(svc.Spec.Ports[0].Port))

	return service
}

func (r *Register) watchPods(ctx context.Context, namespace, selector, serviceName string, port int) {
	for {
		// 简单定期轮询而不是使用K8s watch API
		time.Sleep(r.syncInterval)

		pods, err := r.cli.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
			LabelSelector: selector,
		})
		if err != nil {
			logger.CtxErrorf(ctx, "watch pods from k8s error: %v", err)
			continue
		}

		service := &discov.Service{
			Name:      serviceName,
			Endpoints: make([]*discov.Endpoint, 0),
		}

		for _, pod := range pods.Items {
			if pod.Status.Phase != "Running" {
				continue
			}

			podIP := pod.Status.PodIP
			if podIP == "" {
				continue
			}

			endpoint := &discov.Endpoint{
				ServerName: pod.Name,
				IP:         podIP,
				Port:       port,
				Weight:     1,
				Enable:     true,
			}
			service.Endpoints = append(service.Endpoints, endpoint)
		}

		r.updateDownService(service)
	}
}

func (r *Register) updateDownService(service *discov.Service) {
	r.lock.Lock()
	defer r.lock.Unlock()

	downServices := r.getDownServices()
	downServices[service.Name] = service
	r.downServices.Store(downServices)

	r.NotifyListeners()
}

func (r *Register) getDownServices() map[string]*discov.Service {
	allServices := r.downServices.Load()
	if allServices == nil {
		return make(map[string]*discov.Service)
	}

	return allServices.(map[string]*discov.Service)
}
