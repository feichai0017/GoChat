package gateway

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"

	"github.com/feichai0017/GoChat/common/config"
	"github.com/feichai0017/GoChat/common/crpc"
	"github.com/feichai0017/GoChat/common/tcp"
	"github.com/feichai0017/GoChat/gateway/rpc/client"
	"github.com/feichai0017/GoChat/gateway/rpc/service"
	"google.golang.org/grpc"
)

var cmdChannel chan *service.CmdContext

// RunMain start gateway server
func RunMain(path string) {
	config.Init(path)
	ln, err := net.ListenTCP("tcp", &net.TCPAddr{Port: config.GetGatewayTCPServerPort()})
	if err != nil {
		log.Fatalf("StartTCPEPollServer err:%s", err.Error())
		panic(err)
	}
	initWorkPool()
	initEpoll(ln, runProc)
	fmt.Println("-------------im gateway stated------------")
	cmdChannel = make(chan *service.CmdContext, config.GetGatewayCmdChannelNum())
	s := crpc.NewCServer(
		crpc.WithServiceName(config.GetGatewayServiceName()),
		crpc.WithIP(config.GetGatewayServiceAddr()),
		crpc.WithPort(config.GetGatewayRPCServerPort()), crpc.WithWeight(config.GetGatewayRPCWeight()))
	fmt.Println(config.GetGatewayServiceName(), config.GetGatewayServiceAddr(), config.GetGatewayRPCServerPort(), config.GetGatewayRPCWeight())
	s.RegisterService(func(server *grpc.Server) {
		service.RegisterGatewayServer(server, &service.Service{CmdChannel: cmdChannel})
	})
	// start rpc client
	client.Init()
	// start command handler
	go cmdHandler()
	// start rpc server
	s.Start(context.TODO())
}

func runProc(c *connection, ep *epoller) {
	ctx := context.Background() // initial context
	// step1: read a complete message package
	dataBuf, err := tcp.ReadData(c.conn)
	if err != nil {
		// if the connection is closed when reading the conn, close the connection directly
		// notify state to clean up the status information of the unexpected exit conn
		if errors.Is(err, io.EOF) {
			// this step is asynchronous, does not need to wait for the return success, because the message reliability is guaranteed by the protocol rather than a single cmd
			ep.remove(c)
			client.CancelConn(&ctx, getEndpoint(), c.id, nil)
		}
		return
	}
	err = wPool.Submit(func() {
		// step2: send the message to state server rpc
		client.SendMsg(&ctx, getEndpoint(), c.id, dataBuf)
	})
	if err != nil {
		fmt.Errorf("runProc:err:%+v\n", err.Error())
	}
}

func cmdHandler() {
	for cmd := range cmdChannel {
		// submit the task to the pool asynchronously
		switch cmd.Cmd {
		case service.DelConnCmd:
			wPool.Submit(func() { closeConn(cmd) })
		case service.PushCmd:
			wPool.Submit(func() { sendMsgByCmd(cmd) })
		default:
			panic("command undefined")
		}
	}
}
func closeConn(cmd *service.CmdContext) {
	if connPtr, ok := ep.tables.Load(cmd.ConnID); ok {
		conn, _ := connPtr.(*connection)
		conn.Close()
	}
}
func sendMsgByCmd(cmd *service.CmdContext) {
	if connPtr, ok := ep.tables.Load(cmd.ConnID); ok {
		conn, _ := connPtr.(*connection)
		dp := tcp.DataPgk{
			Len:  uint32(len(cmd.Payload)),
			Data: cmd.Payload,
		}
		tcp.SendData(conn.conn, dp.Marshal())
	}
}

func getEndpoint() string {
	return fmt.Sprintf("%s:%d", config.GetGatewayServiceAddr(), config.GetGatewayRPCServerPort())
}