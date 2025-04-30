package state

import (
	"context"
	"fmt"

	"github.com/feichai0017/GoChat/common/config"
	"github.com/feichai0017/GoChat/common/crpc"
	"google.golang.org/protobuf/proto"

	"github.com/feichai0017/GoChat/common/idl/message"
	"github.com/feichai0017/GoChat/state/rpc/client"
	"github.com/feichai0017/GoChat/state/rpc/service"
	"google.golang.org/grpc"
)

// RunMain start state server
func RunMain(path string) {
	// startup context
	ctx := context.TODO()
	// initialize global configuration
	config.Init(path)
	// initialize RPC client
	client.Init()
	// start time wheel
	InitTimer()
	// start remote cache state machine component
	InitCacheState(ctx)
	// start command processing write coroutine
	go cmdHandler()
	// register rpc server
	s := crpc.NewCServer(
		crpc.WithServiceName(config.GetStateServiceName()),
		crpc.WithIP(config.GetSateServiceAddr()),
		crpc.WithPort(config.GetSateServerPort()), crpc.WithWeight(config.GetSateRPCWeight()))
	s.RegisterService(func(server *grpc.Server) {
		service.RegisterStateServer(server, cs.server)
	})
	// start rpc server
	s.Start(ctx)
}

// consume signal channel, identify the protocol route between gateway and state server
func cmdHandler() {
	for cmdCtx := range cs.server.CmdChannel {
		switch cmdCtx.Cmd {
		case service.CancelConnCmd:
			fmt.Printf("cancel conn endpoint:%s, coonID:%d, data:%+v\n", cmdCtx.Endpoint, cmdCtx.ConnID, cmdCtx.Payload)
			cs.connLogOut(*cmdCtx.Ctx, cmdCtx.ConnID)
		case service.SendMsgCmd:
			msgCmd := &message.MsgCmd{}
			err := proto.Unmarshal(cmdCtx.Payload, msgCmd)
			if err != nil {
				fmt.Printf("SendMsgCmd:err=%s\n", err.Error())
			}
			msgCmdHandler(cmdCtx, msgCmd)
		}
	}
}

// identify message type, identify the protocol route between client and state server
func msgCmdHandler(cmdCtx *service.CmdContext, msgCmd *message.MsgCmd) {
	switch msgCmd.Type {
	case message.CmdType_Login:
		loginMsgHandler(cmdCtx, msgCmd)
	case message.CmdType_Heartbeat:
		hearbeatMsgHandler(cmdCtx, msgCmd)
	case message.CmdType_ReConn:
		reConnMsgHandler(cmdCtx, msgCmd)
	case message.CmdType_UP:
		upMsgHandler(cmdCtx, msgCmd)
	case message.CmdType_ACK:
		ackMsgHandler(cmdCtx, msgCmd)
	}
}

// implement login function
func loginMsgHandler(cmdCtx *service.CmdContext, msgCmd *message.MsgCmd) {
	loginMsg := &message.LoginMsg{}
	err := proto.Unmarshal(msgCmd.Payload, loginMsg)
	if err != nil {
		fmt.Printf("loginMsgHandler:err=%s\n", err.Error())
		return
	}
	if loginMsg.Head != nil {
		// this will send login msg to business layer for processing
		fmt.Println("loginMsgHandler", loginMsg.Head.DeviceID)
	}
	err = cs.connLogin(*cmdCtx.Ctx, loginMsg.Head.DeviceID, cmdCtx.ConnID)
	if err != nil {
		panic(err)
	}
	sendACKMsg(message.CmdType_Login, cmdCtx.ConnID, 0, 0, "login ok")
}

// handle heartbeat message
func hearbeatMsgHandler(cmdCtx *service.CmdContext, msgCmd *message.MsgCmd) {
	heartMsg := &message.HeartbeatMsg{}
	err := proto.Unmarshal(msgCmd.Payload, heartMsg)
	if err != nil {
		fmt.Printf("hearbeatMsgHandler:err=%s\n", err.Error())
		return
	}
	cs.reSetHeartTimer(cmdCtx.ConnID)
	fmt.Printf("hearbeatMsgHandler connID=%d\n", cmdCtx.ConnID)
	// TODO: not reduce communication, can temporarily not reply heartbeat ack
}

// handle re-connection logic
func reConnMsgHandler(cmdCtx *service.CmdContext, msgCmd *message.MsgCmd) {
	reConnMsg := &message.ReConnMsg{}
	err := proto.Unmarshal(msgCmd.Payload, reConnMsg)
	var code uint32
	msg := "reconn ok"
	if err != nil {
		fmt.Printf("reConnMsgHandler:err=%s\n", err.Error())
		return
	}
	// the connID in the re-connection message header is the connID of the last disconnected connection
	if err := cs.reConn(*cmdCtx.Ctx, reConnMsg.Head.ConnID, cmdCtx.ConnID); err != nil {
		code, msg = 1, "reconn failed"
		panic(err)
	}
	sendACKMsg(message.CmdType_ReConn, cmdCtx.ConnID, 0, code, msg)
}

// handle up-stream message, and check message reliability
func upMsgHandler(cmdCtx *service.CmdContext, msgCmd *message.MsgCmd) {
	upMsg := &message.UPMsg{}
	err := proto.Unmarshal(msgCmd.Payload, upMsg)
	if err != nil {
		fmt.Printf("upMsgHandler:err=%s\n", err.Error())
		return
	}
	if cs.compareAndIncrClientID(*cmdCtx.Ctx, cmdCtx.ConnID, upMsg.Head.ClientID) {
		// call downstream business layer rpc, only when the rpc reply is successful can max_clientID be updated
		sendACKMsg(message.CmdType_UP, cmdCtx.ConnID, upMsg.Head.ClientID, 0, "ok")
		// TODO: here should call business layer code
		pushMsg(*cmdCtx.Ctx, cmdCtx.ConnID, cs.msgID, 0, upMsg.UPMsgBody)
	}
}

// handle down-stream message ack reply
func ackMsgHandler(cmdCtx *service.CmdContext, msgCmd *message.MsgCmd) {
	ackMsg := &message.ACKMsg{}
	err := proto.Unmarshal(msgCmd.Payload, ackMsg)
	if err != nil {
		fmt.Printf("ackMsgHandler:err=%s\n", err.Error())
		return
	}
	cs.ackLastMsg(*cmdCtx.Ctx, ackMsg.ConnID, ackMsg.SessionID, ackMsg.MsgID)
}

// called by business layer, handle down-stream message
func pushMsg(ctx context.Context, connID, sessionID, msgID uint64, data []byte) {
	// TODO: first push message here
	pushMsg := &message.PushMsg{
		Content: data,
		MsgID:   cs.msgID,
	}
	if data, err := proto.Marshal(pushMsg); err != nil {
		fmt.Printf("Marshal:err=%s\n", err.Error())
	} else {
		//TODO: here will involve down-stream message sending, whether successfully or not, last msg should be updated
		sendMsg(connID, message.CmdType_Push, data)
		err = cs.appendLastMsg(ctx, connID, pushMsg)
		if err != nil {
			panic(err)
		}
	}
}

// send ack msg
func sendACKMsg(ackType message.CmdType, connID, clientID uint64, code uint32, msg string) {
	ackMsg := &message.ACKMsg{}
	ackMsg.Code = code
	ackMsg.Msg = msg
	ackMsg.ConnID = connID
	ackMsg.Type = ackType
	ackMsg.ClientID = clientID
	downLoad, err := proto.Marshal(ackMsg)
	if err != nil {
		fmt.Println("sendACKMsg", err)
	}
	sendMsg(connID, message.CmdType_ACK, downLoad)
}

// send msg
func sendMsg(connID uint64, ty message.CmdType, downLoad []byte) {
	mc := &message.MsgCmd{}
	mc.Type = ty
	mc.Payload = downLoad
	data, err := proto.Marshal(mc)
	ctx := context.TODO()
	if err != nil {
		fmt.Println("sendMsg", ty, err)
	}
	client.Push(&ctx, connID, data)
}

// re-send push msg
func rePush(connID uint64) {
	pushMsg, err := cs.getLastMsg(context.Background(), connID)
	if err != nil {
		panic(err)
	}
	if pushMsg == nil {
		return
	}
	msgData, err := proto.Marshal(pushMsg)
	if err != nil {
		panic(err)
	}
	sendMsg(connID, message.CmdType_Push, msgData)
	if state, ok := cs.loadConnIDState(connID); ok {
		state.reSetMsgTimer(connID, pushMsg.SessionID, pushMsg.MsgID)
	}
}