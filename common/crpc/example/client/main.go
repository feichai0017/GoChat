package main

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/feichai0017/GoChat/common/config"
	"github.com/feichai0017/GoChat/common/crpc"
	"github.com/feichai0017/GoChat/common/crpc/example/helloservice"
	ptrace "github.com/feichai0017/GoChat/common/crpc/trace"
)

func main() {
	config.Init(currentFileDir() + "/crpc_client.yaml")

	ptrace.StartAgent()
	defer ptrace.StopAgent()

	pCli, _ := crpc.NewCClient("crpc_server")

	ctx, cancel := context.WithTimeout(context.TODO(), 100*time.Second)
	defer cancel()
	cli := helloservice.NewGreeterClient(pCli.Conn())
	resp, err := cli.SayHello(ctx, &helloservice.HelloRequest{
		Name: "xxxxxx",
	})
	fmt.Println(resp, err)
}

func currentFileDir() string {
	_, file, _, ok := runtime.Caller(1)
	parts := strings.Split(file, "/")

	if !ok {
		return ""
	}

	dir := ""
	for i := range len(parts) - 1 {
		dir += "/" + parts[i]
	}

	return dir[1:]
}
