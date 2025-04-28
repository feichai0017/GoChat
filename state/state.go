package state

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/feichai0017/GoChat/common/cache"
	"github.com/feichai0017/GoChat/common/router"
	"github.com/feichai0017/GoChat/common/timingwheel"
	"github.com/feichai0017/GoChat/state/rpc/client"
)

type connState struct {
	sync.RWMutex
	heartTimer   *timingwheel.Timer
	reConnTimer  *timingwheel.Timer
	msgTimer     *timingwheel.Timer
	msgTimerLock string
	connID       uint64
	did          uint64
}

func (c *connState) close(ctx context.Context) error {
	c.Lock()
	defer c.Unlock()
	if c.heartTimer != nil {
		c.heartTimer.Stop()
	}
	if c.reConnTimer != nil {
		c.reConnTimer.Stop()
	}
	if c.msgTimer != nil {
		c.msgTimer.Stop()
	}
	// TODO: how to ensure transactionality here, think about it, or whether it is necessary to ensure
	// TODO: here can also use lua or pipeline to merge two redis operations as much as possible, which is effective in large-scale applications
	// TODO: this is a good thing to think about, the time&space complexity of network calls
	slotKey := cs.getLoginSlotKey(c.connID)
	meta := cs.loginSlotMarshal(c.did, c.connID)
	err := cache.SREM(ctx, slotKey, meta)
	if err != nil {
		return err
	}

	slot := cs.getConnStateSlot(c.connID)

	key := fmt.Sprintf(cache.MaxClientIDKey, slot, c.connID)
	err = cache.Del(ctx, key)
	if err != nil {
		return err
	}

	err = router.DelRecord(ctx, c.did)
	if err != nil {
		return err
	}

	lastMsg := fmt.Sprintf(cache.LastMsgKey, slot, c.connID)
	err = cache.Del(ctx, lastMsg)
	if err != nil {
		return err
	}

	err = client.DelConn(&ctx, c.connID, nil)
	if err != nil {
		return err
	}

	cs.deleteConnIDState(ctx, c.connID)
	return nil
}

func (c *connState) appendMsg(ctx context.Context, key, msgTimerLock string, msgData []byte) {
	c.Lock()
	defer c.Unlock()
	c.msgTimerLock = msgTimerLock
	if c.msgTimer != nil {
		c.msgTimer.Stop()
		c.msgTimer = nil
	}
	// 创建定时器
	t := AfterFunc(100*time.Millisecond, func() {
		rePush(c.connID)
	})
	c.msgTimer = t
	err := cache.SetBytes(ctx, key, msgData, cache.TTL7D)
	if err != nil {
		panic(key)
	}
}

func (c *connState) reSetMsgTimer(connID, sessionID, msgID uint64) {
	c.Lock()
	defer c.Unlock()
	if c.msgTimer != nil {
		c.msgTimer.Stop()
	}
	c.msgTimerLock = fmt.Sprintf("%d_%d", sessionID, msgID)
	c.msgTimer = AfterFunc(100*time.Millisecond, func() {
		rePush(connID)
	})
}

// used to restore when restarting
func (c *connState) loadMsgTimer(ctx context.Context) {
	// create timer
	data, err := cs.getLastMsg(ctx, c.connID)
	if err != nil {
		// the handling here is rough, if online, a more solid solution is needed
		panic(err)
	}
	if data == nil {
		return
	}
	c.reSetMsgTimer(c.connID, data.SessionID, data.MsgID)
}

func (c *connState) reSetHeartTimer() {
	c.Lock()
	defer c.Unlock()
	if c.heartTimer != nil {
		c.heartTimer.Stop()
	}
	c.heartTimer = AfterFunc(5*time.Second, func() {
		c.reSetReConnTimer()
	})
}

func (c *connState) reSetReConnTimer() {
	c.Lock()
	defer c.Unlock()

	if c.reConnTimer != nil {
		c.reConnTimer.Stop()
	}
	
	// initialize re-connection timer
	c.reConnTimer = AfterFunc(10*time.Second, func() {
		ctx := context.TODO()
		// overall connID state logout
		cs.connLogOut(ctx, c.connID)
	})
}

func (c *connState) ackLastMsg(ctx context.Context, sessionID, msgID uint64) bool {
	c.Lock()
	defer c.Unlock()
	msgTimerLock := fmt.Sprintf("%d_%d", sessionID, msgID)
	if c.msgTimerLock != msgTimerLock {
		return false
	}
	slot := cs.getConnStateSlot(c.connID)
	key := fmt.Sprintf(cache.LastMsgKey, slot, c.connID)
	if err := cache.Del(ctx, key); err != nil {
		return false
	}
	if c.msgTimer != nil {
		c.msgTimer.Stop()
	}
	return true
}