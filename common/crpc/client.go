package crpc

import (
	"context"
	"fmt"
	"time"

	"github.com/feichai0017/GoChat/common/crpc/discov/plugin"

	"google.golang.org/grpc/resolver"

	"github.com/feichai0017/GoChat/common/crpc/discov"
	clientinterceptor "github.com/feichai0017/GoChat/common/crpc/interceptor/client"
	presolver "github.com/feichai0017/GoChat/common/crpc/resolver"
	"google.golang.org/grpc"
	"google.golang.org/grpc/balancer/roundrobin"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	dialTimeout = 5 * time.Second
)

type PClient struct {
	serviceName  string
	d            discov.Discovery
	interceptors []grpc.UnaryClientInterceptor
	conn         *grpc.ClientConn
}

// NewPClient ...
func NewPClient(serviceName string, interceptors ...grpc.UnaryClientInterceptor) (*PClient, error) {
	p := &PClient{
		serviceName:  serviceName,
		interceptors: interceptors,
	}

	if p.d == nil {
		dis, err := plugin.GetDiscovInstance()
		if err != nil {
			panic(err)
		}

		p.d = dis
	}

	resolver.Register(presolver.NewDiscovBuilder(p.d))

	conn, err := p.dial()
	p.conn = conn

	return p, err
}

// Conn return *grpc.ClientConn
func (p *PClient) Conn() *grpc.ClientConn {
	return p.conn
}

func (p *PClient) dial() (*grpc.ClientConn, error) {
	svcCfg := fmt.Sprintf(`{"loadBalancingPolicy":"%s"}`, roundrobin.Name)
	balancerOpt := grpc.WithDefaultServiceConfig(svcCfg)

	interceptors := []grpc.UnaryClientInterceptor{
		clientinterceptor.TraceUnaryClientInterceptor(),
		clientinterceptor.MetricUnaryClientInterceptor(),
	}
	interceptors = append(interceptors, p.interceptors...)

	options := []grpc.DialOption{
		balancerOpt,
		grpc.WithChainUnaryInterceptor(interceptors...),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	ctx, _ := context.WithTimeout(context.Background(), dialTimeout)

	return grpc.DialContext(ctx, fmt.Sprintf("discov:///%v", p.serviceName), options...)
}

func (p *PClient) DialByEndPoint(adrss string) (*grpc.ClientConn, error) {
	interceptors := []grpc.UnaryClientInterceptor{
		clientinterceptor.TraceUnaryClientInterceptor(),
		clientinterceptor.MetricUnaryClientInterceptor(),
	}
	interceptors = append(interceptors, p.interceptors...)

	options := []grpc.DialOption{
		grpc.WithChainUnaryInterceptor(interceptors...),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	ctx, _ := context.WithTimeout(context.Background(), dialTimeout)

	return grpc.DialContext(ctx, adrss, options...)
}
