package gateway

import (
	"fmt"
	"log"
	"net"
	"reflect"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"

	"golang.org/x/sys/unix"

	"github.com/feichai0017/GoChat/common/config"
)

// global object
var ep *ePool    // epoll pool
var tcpNum int32 // current service allowed to accept max tcp connections

type ePool struct {
	eChan  chan *connection
	tables sync.Map
	eSize  int
	done   chan struct{}

	ln *net.TCPListener
	f  func(c *connection, ep *epoller)
}

func initEpoll(ln *net.TCPListener, f func(c *connection, ep *epoller)) {
	setLimit()
	ep = newEPool(ln, f)
	ep.createAcceptProcess()
	ep.startEPool()
}

func newEPool(ln *net.TCPListener, cb func(c *connection, ep *epoller)) *ePool {
	return &ePool{
		eChan:  make(chan *connection, config.GetGatewayEpollerChanNum()),
		done:   make(chan struct{}),
		eSize:  config.GetGatewayEpollerNum(),
		tables: sync.Map{},
		ln:     ln,
		f:      cb,
	}
}

// create a dedicated process to handle accept events, corresponding to the number of current cpu cores, to maximize efficiency
func (e *ePool) createAcceptProcess() {
	for range runtime.NumCPU() {
		go func() {
			for {
				conn, e := e.ln.AcceptTCP()
				// rate limiter
				if !checkTcp() {
					_ = conn.Close()
					continue
				}
				setTcpConifg(conn)
				if e != nil {
					if ne, ok := e.(net.Error); ok && ne.Timeout() {
						fmt.Errorf("[ERROR] accept timeout error: %v", ne)
					}
					fmt.Errorf("[ERROR] accept err: %v", e)
				}
				c := NewConnection(conn)
				ep.addTask(c)
			}
		}()
	}
}

func (e *ePool) startEPool() {
	for range e.eSize {
		go e.startEProc()
	}
}

// epoller pool processor
func (e *ePool) startEProc() {
	ep, err := newEpoller()
	if err != nil {
		panic(err)
	}
	// listen connection creation event
	go func() {
		for {
			select {
			case <-e.done:
				return
			case conn := <-e.eChan:
				addTcpNum()
				fmt.Printf("[INFO] tcpNum:%d\n", tcpNum)
				if err := ep.add(conn); err != nil {
					fmt.Printf("[ERROR] failed to add connection %v\n", err)
					conn.Close() // login failed, close connection directly
					continue
				}
				fmt.Printf("[INFO] EpollerPool new connection[%v] tcpSize:%d\n", conn.RemoteAddr(), tcpNum)
			}
		}
	}()
	// epoller here polling to wait, when wait occurs, call the callback function to process
	for {
		select {
		case <-e.done:
			return
		default:
			connections, err := ep.wait(200) // 200ms once polling to avoid busy-waiting
			if err != nil && err != syscall.EINTR {
				fmt.Printf("[ERROR] failed to epoll wait %v\n", err)
				continue
			}
			for _, conn := range connections {
				if conn == nil {
					break
				}
				e.f(conn, ep)
			}
		}
	}
}

func (e *ePool) addTask(c *connection) {
	e.eChan <- c
}

// epoller object
type epoller struct {
	fd            int
	fdToConnTable sync.Map
}

func newEpoller() (*epoller, error) {
	fd, err := unix.EpollCreate1(0)
	if err != nil {
		return nil, err
	}
	return &epoller{
		fd: fd,
	}, nil
}

// TODO: used non-blocking FD, optimize edge-triggered mode
func (e *epoller) add(conn *connection) error {
	// Extract file descriptor associated with the connection
	fd := conn.fd
	// Set socket to non-blocking mode
	if err := unix.SetNonblock(fd, true); err != nil {
		return fmt.Errorf("[ERROR] failed to set socket non-blocking: %v", err)
	}
	// Use Edge-Triggered mode for better performance
	err := unix.EpollCtl(e.fd, syscall.EPOLL_CTL_ADD, fd, &unix.EpollEvent{
		Events: unix.EPOLLIN | unix.EPOLLHUP,
		Fd:     int32(fd),
	})
	if err != nil {
		return err
	}
	e.fdToConnTable.Store(conn.fd, conn)
	ep.tables.Store(conn.id, conn)
	conn.BindEpoller(e)
	return nil
}
func (e *epoller) remove(c *connection) error {
	subTcpNum()
	fd := c.fd
	err := unix.EpollCtl(e.fd, syscall.EPOLL_CTL_DEL, fd, nil)
	if err != nil {
		return err
	}
	ep.tables.Delete(c.id)
	e.fdToConnTable.Delete(c.fd)
	return nil
}
func (e *epoller) wait(msec int) ([]*connection, error) {
	events := make([]unix.EpollEvent, config.GetGatewayEpollWaitQueueSize())
	n, err := unix.EpollWait(e.fd, events, msec)
	if err != nil {
		return nil, err
	}
	var connections []*connection
	for i := range n {
		if conn, ok := e.fdToConnTable.Load(int(events[i].Fd)); ok {
			connections = append(connections, conn.(*connection))
		}
	}
	return connections, nil
}
func socketFD(conn *net.TCPConn) int {
	tcpConn := reflect.Indirect(reflect.ValueOf(*conn)).FieldByName("conn")
	fdVal := tcpConn.FieldByName("fd")
	pfdVal := reflect.Indirect(fdVal).FieldByName("pfd")
	return int(pfdVal.FieldByName("Sysfd").Int())
}

// set the limit of the number of files that the go process can open
func setLimit() {
	var rLimit syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit); err != nil {
		panic(err)
	}
	rLimit.Cur = rLimit.Max
	if err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit); err != nil {
		panic(err)
	}

	log.Printf("set cur limit: %d", rLimit.Cur)
}

func addTcpNum() {
	atomic.AddInt32(&tcpNum, 1)
}

func getTcpNum() int32 {
	return atomic.LoadInt32(&tcpNum)
}
func subTcpNum() {
	atomic.AddInt32(&tcpNum, -1)
}

func checkTcp() bool {
	num := getTcpNum()
	maxTcpNum := config.GetGatewayMaxTcpNum()
	return num <= maxTcpNum
}

func setTcpConifg(c *net.TCPConn) {
	_ = c.SetKeepAlive(true)
}
