package crpc

import (
	"context"
	"testing"

	"github.com/feichai0017/GoChat/common/config"

	"github.com/feichai0017/GoChat/common/crpc/example/helloservice"

	ctrace "github.com/feichai0017/GoChat/common/crpc/trace"
	"google.golang.org/grpc"
)

const (
	testIp   = "127.0.0.1"
	testPort = 8867
)

func TestNewCServer(t *testing.T) {
	config.Init("../../gochat.yaml")

	ctrace.StartAgent()
	defer ctrace.StopAgent()

	s := NewCServer(WithServiceName("crpc_server"), WithIP(testIp), WithPort(testPort), WithWeight(100))
	s.RegisterService(func(server *grpc.Server) {
		helloservice.RegisterGreeterServer(server, helloservice.HelloServer{})
	})
	s.Start(context.TODO())
}
