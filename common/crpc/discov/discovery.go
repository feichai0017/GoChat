package discov

import (
	"context"
)

type Discovery interface {
	// Name discovery name eg etcd zk consul
	Name() string
	// Register register service
	Register(ctx context.Context, service *Service)
	// UnRegister unregister service
	UnRegister(ctx context.Context, service *Service)
	// GetService get service node info
	GetService(ctx context.Context, name string) *Service
	// AddListener add listener
	AddListener(ctx context.Context, f func())
	// NotifyListeners notify all listeners
	NotifyListeners()
}