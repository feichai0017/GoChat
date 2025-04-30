package client

import (
	"context"
	"fmt"
	"time"

	"github.com/feichai0017/GoChat/common/config"
	"github.com/feichai0017/GoChat/common/crpc"
	"github.com/feichai0017/GoChat/state/rpc/service"
)

var stateClient service.StateClient

func initStateClient() {
	pCli, err := crpc.NewCClient(config.GetStateServiceName())
	if err != nil {
		panic(err)
	}
	cli, err := pCli.DialByEndPoint(config.GetGatewayStateServerEndPoint())
	if err != nil {
		panic(err)
	}
	stateClient = service.NewStateClient(cli)
}

func CancelConn(ctx *context.Context, endpoint string, connID uint64, Payload []byte) error {
	rpcCtx, cancel := context.WithTimeout(*ctx, 100*time.Millisecond)
	defer cancel()
	stateClient.CancelConn(rpcCtx, &service.StateRequest{
		Endpoint: endpoint,
		ConnID:   connID,
		Data:     Payload,
	})
	return nil
}

func SendMsg(ctx *context.Context, endpoint string, connID uint64, Payload []byte) error {
	rpcCtx, cancel := context.WithTimeout(*ctx, 100*time.Millisecond)
	defer cancel()

	fmt.Println("sendMsg", connID, string(Payload))
	_, err := stateClient.SendMsg(rpcCtx, &service.StateRequest{
		Endpoint: endpoint,
		ConnID:   connID,
		Data:     Payload,
	})
	if err != nil {
		fmt.Println("sendMsg error", err)
		panic(err)
	}
	return nil
}