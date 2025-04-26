package ipconf

import (
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/feichai0017/GoChat/common/config"
	"github.com/feichai0017/GoChat/ipconf/domain"
	"github.com/feichai0017/GoChat/ipconf/source"
)


func RunMain(path string) {
	config.Init(path)
	source.Init()
	domain.Init()
	s := server.Default(server.WithHostPorts(":6789"))
	s.GET("/ip/list", GetIpInfoList)
	s.Spin()
}
