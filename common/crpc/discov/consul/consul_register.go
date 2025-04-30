package consul

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bytedance/gopkg/util/logger"
	"github.com/feichai0017/GoChat/common/crpc/discov"
	"github.com/hashicorp/consul/api"
)

const KeyPrefix = "gochat/crpc/"

// Register ...
type Register struct {
	Options
	cli                 *api.Client
	serviceRegisterCh   chan *discov.Service
	serviceUnRegisterCh chan *discov.Service
	lock                sync.Mutex
	downServices        atomic.Value
	registerServices    map[string]*registerService
	listeners           []func()
}

type registerService struct {
	service      *discov.Service
	isRegistered bool
}

// NewConsulRegister ...
func NewConsulRegister(opts ...Option) (discov.Discovery, error) {
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
		registerServices:    make(map[string]*registerService),
	}

	if err := r.init(context.TODO()); err != nil {
		return nil, err
	}

	return r, nil
}

// Init 初始化
func (r *Register) init(ctx context.Context) error {
	config := api.DefaultConfig()
	if len(r.endpoints) > 0 {
		parts := strings.Split(r.endpoints[0], ":")
		config.Address = parts[0]
		if len(parts) > 1 {
			port, _ := strconv.Atoi(parts[1])
			if port > 0 {
				config.Address = fmt.Sprintf("%s:%d", parts[0], port)
			}
		}
	}

	var err error
	r.cli, err = api.NewClient(config)
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
			if _, ok := r.registerServices[service.Name]; ok {
				r.registerServices[service.Name].service.Endpoints = append(r.registerServices[service.Name].service.Endpoints, service.Endpoints...)
				r.registerServices[service.Name].isRegistered = false // 重新上报到consul
			} else {
				r.registerServices[service.Name] = &registerService{
					service:      service,
					isRegistered: false,
				}
			}
		case service := <-r.serviceUnRegisterCh:
			if _, ok := r.registerServices[service.Name]; !ok {
				logger.CtxErrorf(context.TODO(), "UnRegisterService err, service %v was not registered", service.Name)
				continue
			}
			r.unRegisterService(context.TODO(), service)
		default:
			r.registerAllServices(context.TODO())
			time.Sleep(r.registerServiceInterval)
		}
	}
}

func (r *Register) registerAllServices(ctx context.Context) {
	for _, service := range r.registerServices {
		if !service.isRegistered {
			r.registerService(ctx, service)
			r.registerServices[service.service.Name].isRegistered = true
		}
	}
}

func (r *Register) registerService(ctx context.Context, service *registerService) {
	for _, endpoint := range service.service.Endpoints {
		serviceID := r.getConsulServiceID(service.service.Name, endpoint.IP, endpoint.Port)

		registration := &api.AgentServiceRegistration{
			ID:      serviceID,
			Name:    service.service.Name,
			Address: endpoint.IP,
			Port:    endpoint.Port,
			Tags:    []string{"gochat", "crpc"},
			Check: &api.AgentServiceCheck{
				HTTP:     fmt.Sprintf("http://%s:%d/health", endpoint.IP, endpoint.Port),
				Interval: "10s",
				Timeout:  "3s",
			},
		}

		if err := r.cli.Agent().ServiceRegister(registration); err != nil {
			logger.CtxErrorf(ctx, "register service err,err:%v, register data:%v", err, serviceID)
			continue
		}
	}

	service.isRegistered = true
}

func (r *Register) unRegisterService(ctx context.Context, service *discov.Service) {
	endpoints := make([]*discov.Endpoint, 0)
	for _, endpoint := range r.registerServices[service.Name].service.Endpoints {
		var isRemove bool
		for _, unRegisterEndpoint := range service.Endpoints {
			if endpoint.IP == unRegisterEndpoint.IP && endpoint.Port == unRegisterEndpoint.Port {
				serviceID := r.getConsulServiceID(service.Name, endpoint.IP, endpoint.Port)
				err := r.cli.Agent().ServiceDeregister(serviceID)
				if err != nil {
					logger.CtxErrorf(ctx, "UnRegisterService consul del err, service %v was not registered", service.Name)
				}
				isRemove = true
				break
			}
		}

		if !isRemove {
			endpoints = append(endpoints, endpoint)
		}
	}

	if len(endpoints) == 0 {
		delete(r.registerServices, service.Name)
	} else {
		r.registerServices[service.Name].service.Endpoints = endpoints
	}
}

func (r *Register) Name() string {
	return "consul"
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
	r.serviceRegisterCh <- service
}

func (r *Register) UnRegister(ctx context.Context, service *discov.Service) {
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

	services, _, err := r.cli.Health().Service(name, "", true, nil)
	if err != nil {
		logger.CtxErrorf(ctx, "get service from consul error: %v", err)
		return &discov.Service{
			Name:      name,
			Endpoints: make([]*discov.Endpoint, 0),
		}
	}

	service := &discov.Service{
		Name:      name,
		Endpoints: make([]*discov.Endpoint, 0),
	}

	for _, s := range services {
		endpoint := &discov.Endpoint{
			ServerName: s.Service.Service,
			IP:         s.Service.Address,
			Port:       s.Service.Port,
			Weight:     1,
			Enable:     true,
		}
		service.Endpoints = append(service.Endpoints, endpoint)
	}

	allServices[name] = service
	r.downServices.Store(allServices)

	go r.watchService(ctx, name)

	return service
}

func (r *Register) watchService(ctx context.Context, name string) {
	var waitIndex uint64 = 0
	for {
		services, meta, err := r.cli.Health().Service(name, "", true, &api.QueryOptions{
			WaitIndex: waitIndex,
		})
		if err != nil {
			logger.CtxErrorf(ctx, "watch service from consul error: %v", err)
			time.Sleep(time.Second)
			continue
		}

		waitIndex = meta.LastIndex

		service := &discov.Service{
			Name:      name,
			Endpoints: make([]*discov.Endpoint, 0),
		}

		for _, s := range services {
			endpoint := &discov.Endpoint{
				ServerName: s.Service.Service,
				IP:         s.Service.Address,
				Port:       s.Service.Port,
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

func (r *Register) getConsulServiceID(name, ip string, port int) string {
	return fmt.Sprintf("%s-%s-%d", name, ip, port)
}
