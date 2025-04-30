package client

import (
	"context"
	"fmt"
	"time"

	"github.com/feichai0017/GoChat/common/config"
	"github.com/feichai0017/GoChat/common/crpc"
	"github.com/feichai0017/GoChat/gateway/rpc/service"
)

var gatewayClient service.GatewayClient

func initGatewayClient() {
	pCli, err := crpc.NewCClient(config.GetGatewayServiceName())
	if err != nil {
		panic(err)
	}
	conn, err := pCli.DialByEndPoint(config.GetStateServerGatewayServerEndpoint())
	if err != nil {
		panic(err)
	}
	gatewayClient = service.NewGatewayClient(conn)
}

func DelConn(ctx *context.Context, connID uint64, Payload []byte) error {
	rpcCtx, cancel := context.WithTimeout(*ctx, 100*time.Millisecond)
	defer cancel()
	gatewayClient.DelConn(rpcCtx, &service.GatewayRequest{ConnID: connID, Data: Payload})
	return nil
}

func Push(ctx *context.Context, connID uint64, Payload []byte) error {
	rpcCtx, cancel := context.WithTimeout(*ctx, 100*time.Second)
	defer cancel()
	resp, err := gatewayClient.Push(rpcCtx, &service.GatewayRequest{ConnID: connID, Data: Payload})
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(resp)
	return nil
}