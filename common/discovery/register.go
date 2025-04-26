package discovery

import (
	"context"
	"log"

	"github.com/bytedance/gopkg/util/logger"
	"github.com/feichai0017/GoChat/common/config"
	"go.etcd.io/etcd/client/v3"
)

// Make ServiceRegister generic
type ServiceRegister[T any] struct {
	cli           *clientv3.Client
	leaseID       clientv3.LeaseID
	keepAliveChan <-chan *clientv3.LeaseKeepAliveResponse
	key           string
	val           string // Keep val as string (marshalled JSON)
	ctx           *context.Context
}

func NewServiceRegister[T any](ctx *context.Context, key string, endpointInfo *EndpointInfo[T], lease int64) (*ServiceRegister[T], error) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   config.GetEndpointsForDiscovery(),
		DialTimeout: config.GetTimeoutForDiscovery(),
	})
	if err != nil {
		log.Fatal(err)
	}

	ser := &ServiceRegister[T]{
		cli: cli,
		key: key,
		val: endpointInfo.Marshal(),
		ctx: ctx,
	}
	// apply lease and set time keepalive
	if err := ser.putKeyWithLease(lease); err != nil {
		return nil, err
	}

	return ser, nil
}

// Update receiver
func (s *ServiceRegister[T]) putKeyWithLease(lease int64) error {
	resp, err := s.cli.Grant(*s.ctx, lease)
	if err != nil {
		return err
	}

	_, err = s.cli.Put(*s.ctx, s.key, s.val, clientv3.WithLease(resp.ID))
	if err != nil {
		return err
	}

	leaseRespChan, err := s.cli.KeepAlive(*s.ctx, resp.ID)
	if err != nil {
		return err
	}

	s.leaseID = resp.ID
	s.keepAliveChan = leaseRespChan

	return nil
}

// Make UpdateValue generic
func (s *ServiceRegister[T]) UpdateValue(val *EndpointInfo[T]) error {
	value := val.Marshal()
	_, err := s.cli.Put(*s.ctx, s.key, value, clientv3.WithLease(s.leaseID))
	if err != nil {
		return err
	}
	s.val = value
	logger.CtxInfof(*s.ctx, "ServiceRegister.updateValue leaseID=%d Put key=%s,val=%s, success!", s.leaseID, s.key, s.val)
	return nil
}

// Update receiver
func (s *ServiceRegister[T]) ListenLeaseRespChan() {
	for leaseKeepResp := range s.keepAliveChan {
		logger.CtxInfof(*s.ctx, "lease success leaseID:%d, Put key:%s,val:%s reps:+%v", s.leaseID, s.key, s.val, leaseKeepResp)
	}
	logger.CtxInfof(*s.ctx, "lease failed !!!  leaseID:%d, Put key:%s,val:%s", s.leaseID, s.key, s.val)
}

// Update receiver
func (s *ServiceRegister[T]) Close() error {
	if _, err := s.cli.Revoke(context.Background(), s.leaseID); err != nil {
		return err
	}
	logger.CtxInfof(*s.ctx, "lease close !!!  leaseID:%d, Put key:%s,val:%s  success!", s.leaseID, s.key, s.val)
	return s.cli.Close()
}
