package plugin

import (
	"fmt"

	"github.com/feichai0017/GoChat/common/crpc/config"
	"github.com/feichai0017/GoChat/common/crpc/discov"
	"github.com/feichai0017/GoChat/common/crpc/discov/consul"
	"github.com/feichai0017/GoChat/common/crpc/discov/etcd"
	"github.com/feichai0017/GoChat/common/crpc/discov/k8s"
)

// GetDiscovInstance get discov instance
func GetDiscovInstance() (discov.Discovery, error) {
	name := config.GetDiscovName()
	switch name {
	case "etcd":
		return etcd.NewETCDRegister(etcd.WithEndpoints(config.GetDiscovEndpoints()))
	case "consul":
		return consul.NewConsulRegister(consul.WithEndpoints(config.GetDiscovEndpoints()))
	case "k8s":
		return k8s.NewK8sRegister()
	}

	return nil, fmt.Errorf("not exist plugin:%s", name)
}
