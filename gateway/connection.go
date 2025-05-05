package gateway

import (
	"errors"
	"net"
	"sync"
	"time"
)

var node *ConnIDGenerater

const (
	version      = uint64(0) // version control
	sequenceBits = uint64(16)

	maxSequence = int64(-1) ^ (int64(-1) << sequenceBits)

	timeLeft    = uint8(16) // timeLeft = sequenceBits // timeLeft to the left shift
	versionLeft = uint8(63) // move to the highest bit
	// 2022-11-25 00:00:00 +0800 CST
	twepoch = int64(1669334400000) // constant timestamp (millisecond)
)

type ConnIDGenerater struct {
	mu        sync.Mutex
	LastStamp int64 // record the last ID timestamp
	Sequence  int64 // current ID sequence number generated in 1 millisecond (from 0 to start)
}

type connection struct {
	id   uint64 // unique ID
	fd   int
	e    *epoller
	conn *net.TCPConn
}

func init() {
	node = &ConnIDGenerater{}
}

func (c *ConnIDGenerater) getMilliSeconds() int64 {
	return time.Now().UnixNano() / 1e6
}

func NewConnection(conn *net.TCPConn) *connection {
	var id uint64
	var err error
	if id, err = node.NextID(); err != nil {
		panic(err) // online service needs to solve this problem, panic instead of error
	}
	return &connection{
		id:   id,
		fd:   socketFD(conn),
		conn: conn,
	}
}
func (c *connection) Close() {
	ep.tables.Delete(c.id)
	if c.e != nil {
		c.e.fdToConnTable.Delete(c.fd)
	}
	err := c.conn.Close()
	panic(err)
}

func (c *connection) RemoteAddr() string {
	return c.conn.RemoteAddr().String()
}

func (c *connection) BindEpoller(e *epoller) {
	c.e = e
}

// The lock will spin, but it will not affect performance much, mainly because the critical area is small
func (w *ConnIDGenerater) NextID() (uint64, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.nextID()
}

func (w *ConnIDGenerater) nextID() (uint64, error) {
	timeStamp := w.getMilliSeconds()
	if timeStamp < w.LastStamp {
		return 0, errors.New("time is moving backwards,waiting until")
	}

	if w.LastStamp == timeStamp {
		w.Sequence = (w.Sequence + 1) & maxSequence
		if w.Sequence == 0 { // if there is overflow here, then wait until the next millisecond to allocate, so there will be no repetition
			for timeStamp <= w.LastStamp {
				timeStamp = w.getMilliSeconds()
			}
		}
	} else { // if the timestamp is not equal to the last time, then in order to prevent possible clock drift, it must be re-counted
		w.Sequence = 0
	}
	w.LastStamp = timeStamp
	// subtract to compress the timestamp
	id := ((timeStamp - twepoch) << timeLeft) | w.Sequence
	connID := uint64(id) | (version << versionLeft)
	return connID, nil
}