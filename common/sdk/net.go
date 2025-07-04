package sdk

import (
	"encoding/json"
	"fmt"
	"net"
	"sync/atomic"

	"google.golang.org/protobuf/proto"

	"github.com/feichai0017/GoChat/common/idl/message"
	"github.com/feichai0017/GoChat/common/tcp"
)

type connect struct {
	sendChan, recvChan chan *Message
	conn               *net.TCPConn
	connID             uint64
	ip                 net.IP
	port               int
}

func newConnet(ip net.IP, port int) *connect {
	clientConn := &connect{
		sendChan: make(chan *Message),
		recvChan: make(chan *Message),
		ip:       ip,
		port:     port,
	}
	addr := &net.TCPAddr{IP: ip, Port: port}
	conn, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		fmt.Printf("[ERROR] DialTCP.err=%+v", err)
		return nil
	}
	clientConn.conn = conn
	return clientConn
}

func handAckMsg(c *connect, data []byte) *Message {
	ackMsg := &message.ACKMsg{}
	proto.Unmarshal(data, ackMsg)
	switch ackMsg.Type {
	case message.CmdType_Login, message.CmdType_ReConn:
		atomic.StoreUint64(&c.connID, ackMsg.ConnID)
	}
	return &Message{
		Type:       MsgTypeAck,
		Name:       "gochat",
		FormUserID: "1212121",
		ToUserID:   "222212122",
		Content:    ackMsg.Msg,
	}
}
func handPushMsg(c *connect, data []byte) *Message {
	pushMsg := &message.PushMsg{}
	proto.Unmarshal(data, pushMsg)
	// if pushMsg.MsgID == c.maxMsgID+1 {
	// 	c.maxMsgID++
	msg := &Message{}
	json.Unmarshal(pushMsg.Content, msg)
	ackMsg := &message.ACKMsg{
		Type:   message.CmdType_UP,
		ConnID: c.connID,
	}
	ackData, _ := proto.Marshal(ackMsg)
	c.send(message.CmdType_ACK, ackData)
	return msg
	// }
}

func (c *connect) reConn() {
	c.conn.Close()
	addr := &net.TCPAddr{IP: c.ip, Port: c.port}
	conn, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		fmt.Printf("[ERROR] DialTCP.err=%+v", err)
	}
	c.conn = conn
}
func (c *connect) send(ty message.CmdType, palyload []byte) error {
	// Directly send to receiver
	msgCmd := message.MsgCmd{
		Type:    ty,
		Payload: palyload,
	}
	msg, err := proto.Marshal(&msgCmd)
	if err != nil {
		panic(err)
	}
	dataPgk := tcp.DataPgk{
		Data: msg,
		Len:  uint32(len(msg)),
	}
	_, err = c.conn.Write(dataPgk.Marshal())
	return err
}

func (c *connect) recv() <-chan *Message {
	return c.recvChan
}

func (c *connect) close() {
	// Nothing to recycle for now
	c.conn.Close()
}
