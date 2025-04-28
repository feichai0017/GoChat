package plugin

import (
	"errors"
	"fmt"

	"github.com/feichai0017/GoChat/common/crpc/config"
	"github.com/feichai0017/GoChat/common/crpc/discov"
	"github.com/feichai0017/GoChat/common/crpc/discov/etcd"
)

// GetDiscovInstance get discov instance
func GetDiscovInstance() (discov.Discovery, error) {
	name := config.GetDiscovName()
	switch name {
	case "etcd":
		return etcd.NewETCDRegister(etcd.WithEndpoints(config.GetDiscovEndpoints()))
	}

	return nil, errors.New(fmt.Sprintf("not exist plugin:%s", name))
}