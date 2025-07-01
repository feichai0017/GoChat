package gateway

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"syscall"

	"google.golang.org/grpc"

	"github.com/feichai0017/GoChat/common/config"
	"github.com/feichai0017/GoChat/common/crpc"
	"github.com/feichai0017/GoChat/common/tcp"
	"github.com/feichai0017/GoChat/gateway/rpc/client"
	"github.com/feichai0017/GoChat/gateway/rpc/service"
)

var cmdChannel chan *service.CmdContext

// RunMain start gateway server
func RunMain(path string) {
	config.Init(path)
	ln, err := net.ListenTCP("tcp", &net.TCPAddr{Port: config.GetGatewayTCPServerPort()})
	if err != nil {
		log.Fatalf("[FATAL] StartTCPEPollServer err:%s", err.Error())
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

	// Start a loop, because ET mode requires reading all data at once
	for {
		// Create a temporary buffer to read data from the socket
		tempBuf := make([]byte, 4096)
		n, err := c.conn.Read(tempBuf)

		if n > 0 {
			// Write the received data into the dedicated buffer for this connection
			c.readBuf.Write(tempBuf[:n])
		}

		if err != nil {
			// EAGAIN or EWOULDBLOCK means kernel buffer has been fully read, this is the normal exit condition in ET mode
			if errors.Is(err, syscall.EAGAIN) || errors.Is(err, syscall.EWOULDBLOCK) {
				break // Exit the read loop
			}

			// EOF or other errors mean the connection is closed or has an error
			if errors.Is(err, io.EOF) {
				// Connection closea
				fmt.Printf("[ERROR] Connection %d closed with error: %v", c.id, err)
				ctx := context.Background()
				ep.remove(c)
				client.CancelConn(&ctx, getEndpoint(), c.id, nil)
			}
			
			return // Stop handling this connection
		}

		// If a single Read fills the buffer, there may still be data remaining, so continue reading
		if n < len(tempBuf) {
			break // Kernel buffer has been fully read
		}
	}

	// After all read operations are complete, perform centralized packet parsing and forwarding for the buffer
	parseAndForward(c)
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
func parseAndForward(c *connection) {
	for {
		// Packet header length is 4 bytes (uint32)
		if c.readBuf.Len() < 4 {
			break // Not enough data in buffer for a complete header, exit
		}

		headerBytes := c.readBuf.Bytes()[:4]
		var dataLen uint32
		// Use binary.Read to parse the length from the byte stream
		binary.Read(bytes.NewReader(headerBytes), binary.BigEndian, &dataLen)

		if uint32(c.readBuf.Len()) < 4+dataLen {
			break // Not enough data for a complete packet, wait for next read
		}

		// Skip the 4-byte header that has already been read
		c.readBuf.Next(4)
		// Read the packet body
		fullMessage := make([]byte, dataLen)
		c.readBuf.Read(fullMessage)

		// Asynchronously submit to the worker pool for processing
		wPool.Submit(func() {
			ctx := context.Background()
			client.SendMsg(&ctx, getEndpoint(), c.id, fullMessage)
		})
	}
}
