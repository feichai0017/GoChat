package gateway

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"

	"github.com/feichai0017/GoChat/common/config"
	"github.com/feichai0017/GoChat/common/tcp"
)

// RunMain start gateway server
func RunMain(path string) {
	config.Init(path)
	ln, err := net.ListenTCP("tcp", &net.TCPAddr{Port: config.GetGatewayServerPort()})
	if err != nil {
		log.Fatalf("StartTCPEPollServer err:%s", err.Error())
		panic(err)
	}
	initWorkPool()
	initEpoll(ln, runProc)
	fmt.Println("-------------im gateway stated------------")
	select {}
}

func runProc(c *connection, ep *epoller) {

	// step1: read a complete message package
	dataBuf, err := tcp.ReadData(c.conn)
	if err != nil {

		// if read conn found connection closed, then close connection
		if errors.Is(err, io.EOF) {
			ep.remove(c)
		}
		return
	}
	err = wPool.Submit(func() {
		// step2: submit to state server rpc process
		bytes := tcp.DataPgk{
			Len:  uint32(len(dataBuf)),
			Data: dataBuf,
		}
		tcp.SendData(c.conn, bytes.Marshal())
	})
	if err != nil {
		fmt.Errorf("runProc:err:%+v\n", err.Error())
	}
}