package crpc

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/feichai0017/GoChat/common/crpc/discov/plugin"

	"github.com/bytedance/gopkg/util/logger"

	"github.com/feichai0017/GoChat/common/crpc/discov"
	serverinterceptor "github.com/feichai0017/GoChat/common/crpc/interceptor/server"
	"google.golang.org/grpc"
)

type RegisterFn func(*grpc.Server)

type CServer struct {
	serverOptions
	registers    []RegisterFn
	interceptors []grpc.UnaryServerInterceptor
}

type serverOptions struct {
	serviceName string
	ip          string
	port        int
	weight      int
	health      bool
	d           discov.Discovery
}

type ServerOption func(opts *serverOptions)

// WithServiceName set serviceName
func WithServiceName(serviceName string) ServerOption {
	return func(opts *serverOptions) {
		opts.serviceName = serviceName
	}
}

// WithIP set ip
func WithIP(ip string) ServerOption {
	return func(opts *serverOptions) {
		opts.ip = ip
	}
}

// WithPort set port
func WithPort(port int) ServerOption {
	return func(opts *serverOptions) {
		opts.port = port
	}
}

// WithWeight set weight
func WithWeight(weight int) ServerOption {
	return func(opts *serverOptions) {
		opts.weight = weight
	}
}

// WithHealth set health
func WithHealth(health bool) ServerOption {
	return func(opts *serverOptions) {
		opts.health = health
	}
}

func NewCServer(opts ...ServerOption) *CServer {
	opt := serverOptions{}
	for _, o := range opts {
		o(&opt)
	}

	if opt.d == nil {
		dis, err := plugin.GetDiscovInstance()
		if err != nil {
			panic(err)
		}

		opt.d = dis
	}

	return &CServer{
		opt,
		make([]RegisterFn, 0),
		make([]grpc.UnaryServerInterceptor, 0),
	}
}

// RegisterService ...
// eg :
// p.RegisterService(func(server *grpc.Server) {
//     test.RegisterGreeterServer(server, &Server{})
// })
func (p *CServer) RegisterService(register ...RegisterFn) {
	p.registers = append(p.registers, register...)
}

// RegisterUnaryServerInterceptor register custom interceptor
func (p *CServer) RegisterUnaryServerInterceptor(i grpc.UnaryServerInterceptor) {
	p.interceptors = append(p.interceptors, i)
}

// Start start server
func (p *CServer) Start(ctx context.Context) {
	service := discov.Service{
		Name: p.serviceName,
		Endpoints: []*discov.Endpoint{
			{
				ServerName: p.serviceName,
				IP:         p.ip,
				Port:       p.port,
				Weight:     p.weight,
				Enable:     true,
			},
		},
	}

	// load middleware
	interceptors := []grpc.UnaryServerInterceptor{
		serverinterceptor.RecoveryUnaryServerInterceptor(),
		serverinterceptor.TraceUnaryServerInterceptor(),
		serverinterceptor.MetricUnaryServerInterceptor(p.serviceName),
	}
	interceptors = append(interceptors, p.interceptors...)

	s := grpc.NewServer(grpc.ChainUnaryInterceptor(interceptors...))

	// register service
	for _, register := range p.registers {
		register(s)
	}

	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", p.ip, p.port))
	if err != nil {
		panic(err)
	}

	go func() {
		if err := s.Serve(lis); err != nil {
			panic(err)
		}
	}()
	// register service
	p.d.Register(ctx, &service)

	logger.Info("start CRPC success")

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)
	for {
		sig := <-c
		switch sig {
		case syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT:
			s.Stop()
			p.d.UnRegister(ctx, &service)
			time.Sleep(time.Second)
			return
		case syscall.SIGHUP:
		default:
			return
		}
	}

}
