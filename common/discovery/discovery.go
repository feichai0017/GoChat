package discovery

import (
	"context"
	"sync"

	"github.com/bytedance/gopkg/util/logger"
	"github.com/feichai0017/GoChat/common/config"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type ServiceDiscovery struct {
	cli  *clientv3.Client
	lock sync.RWMutex
	ctx  *context.Context
}

func NewServiceDiscovery(ctx *context.Context) *ServiceDiscovery {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   config.GetEndpointsForDiscovery(),
		DialTimeout: config.GetTimeoutForDiscovery(),
	})
	if err != nil {
		logger.Fatal(err)
	}
	return &ServiceDiscovery{
		cli: cli,
		ctx: ctx,
	}
}

// WatchService initializes the service discovery and watches for changes
func (s *ServiceDiscovery) WatchService(prefix string, set, del func(key, value string)) error {
	resp, err := s.cli.Get(*s.ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return err
	}
	for _, kv := range resp.Kvs {
		set(string(kv.Key), string(kv.Value))
	}

	s.watcher(prefix, set, del)
	return nil
}

// watcher watches for prefix changes in the service discovery
func (s *ServiceDiscovery) watcher(prefix string, set, del func(key, value string)) {
	rch := s.cli.Watch(*s.ctx, prefix, clientv3.WithPrefix())
	logger.CtxInfof(*s.ctx, "Watching prefix: %v now...", prefix)
	for wresp := range rch {
		for _, ev := range wresp.Events {
			switch ev.Type {
			case mvccpb.PUT:
				set(string(ev.Kv.Key), string(ev.Kv.Value))
			case mvccpb.DELETE:
				del(string(ev.Kv.Key), string(ev.Kv.Value))
			}
		}
	}
}

func (s *ServiceDiscovery) Close() error {
	return s.cli.Close()
}
