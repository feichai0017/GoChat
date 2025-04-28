package main

import (
	"context"
	"runtime"
	"strings"

	"github.com/feichai0017/GoChat/common/config"

	"github.com/feichai0017/GoChat/common/crpc"
	"github.com/feichai0017/GoChat/common/crpc/example/helloservice"
	ptrace "github.com/feichai0017/GoChat/common/crpc/trace"
	"google.golang.org/grpc"
)

const (
	testIp   = "127.0.0.1"
	testPort = 8867
)

func main() {
	config.Init(currentFileDir() + "/crpc_server.yaml")

	ptrace.StartAgent()
	defer ptrace.StopAgent()

	s := crpc.NewPServer(crpc.WithServiceName("crpc_server"), crpc.WithIP(testIp), crpc.WithPort(testPort), crpc.WithWeight(100))
	s.RegisterService(func(server *grpc.Server) {
		helloservice.RegisterGreeterServer(server, helloservice.HelloServer{})
	})
	s.Start(context.TODO())
}

func currentFileDir() string {
	_, file, _, ok := runtime.Caller(1)
	parts := strings.Split(file, "/")

	if !ok {
		return ""
	}

	dir := ""
	for i := 0; i < len(parts)-1; i++ {
		dir += "/" + parts[i]
	}

	return dir[1:]
}
