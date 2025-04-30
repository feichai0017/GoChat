package crpc

import (
	"context"
	"fmt"
	"time"

	"github.com/feichai0017/GoChat/common/crpc/discov/plugin"

	"google.golang.org/grpc/connectivity"
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

type CClient struct {
	serviceName  string
	d            discov.Discovery
	interceptors []grpc.UnaryClientInterceptor
	conn         *grpc.ClientConn
}

// NewCClient ...
func NewCClient(serviceName string, interceptors ...grpc.UnaryClientInterceptor) (*CClient, error) {
	p := &CClient{
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
func (p *CClient) Conn() *grpc.ClientConn {
	return p.conn
}

func (p *CClient) dial() (*grpc.ClientConn, error) {
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
		grpc.WithChainUnaryInterceptor(interceptors...),
		grpc.WithDefaultServiceConfig(svcCfg),
	}

	conn, err := grpc.NewClient(
		fmt.Sprintf("discov:///%v", p.serviceName),
		options...,
	)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func (p *CClient) DialByEndPoint(address string) (*grpc.ClientConn, error) {
	interceptors := []grpc.UnaryClientInterceptor{
		clientinterceptor.TraceUnaryClientInterceptor(),
		clientinterceptor.MetricUnaryClientInterceptor(),
	}
	interceptors = append(interceptors, p.interceptors...)

	options := []grpc.DialOption{
		grpc.WithChainUnaryInterceptor(interceptors...),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}


	conn, err := grpc.NewClient(address, options...)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

// waitForReady check if the connection is ready
func waitForReady(conn *grpc.ClientConn, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	for {
		state := conn.GetState()
		if state == connectivity.Ready {
			return nil // success
		}

		if !conn.WaitForStateChange(ctx, state) {
			return fmt.Errorf("gRPC connection not ready within %v (last state: %s)", timeout, state.String())
		}
	}
}