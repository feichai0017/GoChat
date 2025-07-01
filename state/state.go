package state

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/feichai0017/GoChat/common/cache"
	"github.com/feichai0017/GoChat/common/config"
	"github.com/feichai0017/GoChat/common/timingwheel"
	"github.com/feichai0017/GoChat/state/rpc/client"
	"github.com/redis/go-redis/v9"
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

	// 1. Stop local Go timers.
	if c.heartTimer != nil {
		c.heartTimer.Stop()
	}
	if c.reConnTimer != nil {
		c.reConnTimer.Stop()
	}
	if c.msgTimer != nil {
		c.msgTimer.Stop()
	}
	// 2. Atomically clean up all distributed states using a single Lua script.
	// This replaces multiple individual Redis calls.
	slotSize := uint64(len(config.GetStateServerLoginSlotRange()))
	_, err := cache.RunLua(ctx, cache.LuaCleanupConnection, nil, c.connID, c.did, slotSize)
	
	if err != nil && err != redis.Nil {
		// Log a critical error, as this could lead to residual state in Redis.
		// logger.ErrorCtx(ctx, "Failed to cleanup connection state atomically via Lua", "connID", c.connID, "err", err)
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
	// create new timer
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