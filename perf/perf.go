package perf

import (
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/feichai0017/GoChat/common/sdk"
)

// TestMode defines the test mode
type TestMode string

const (
	ConnectMode  TestMode = "connect" // Test connections only
	SendMode     TestMode = "send"    // Test sending upstream messages
	ReceiveMode  TestMode = "receive" // Test receiving downstream messages
	FullTestMode TestMode = "full"    // Complete test (connect+send+receive)
)

// TestConfig test configuration
type TestConfig struct {
	Mode          TestMode // Test mode
	ConnectionNum int32    // Number of connections
	MessageNum    int32    // Number of messages per connection
	MessageSize   int32    // Message size (bytes)
	Interval      int      // Interval between messages (ms)
	Duration      int      // Test duration (s), only for receive mode
	ServerIP      string   // Server IP
	ServerPort    int      // Server port
	Username      string   // Username
	TargetUser    string   // Message recipient
}

// TestResult test results
type TestResult struct {
	ConnSuccess   int32         // Successful connections
	ConnFail      int32         // Failed connections
	MsgSent       int32         // Messages sent
	MsgReceived   int32         // Messages received
	TotalDuration time.Duration // Total test duration
}

var (
	TcpConnNum int32
	result     TestResult
	defaultCfg = TestConfig{
		Mode:          ConnectMode,
		ConnectionNum: 100,
		MessageNum:    10,
		MessageSize:   128,
		Interval:      100,
		Duration:      60,
		ServerIP:      "127.0.0.1",
		ServerPort:    8900,
		Username:      "perf_tester",
		TargetUser:    "perf_receiver",
	}
)

// SetConfig sets the test configuration
func SetConfig(cfg TestConfig) {
	if cfg.ConnectionNum > 0 {
		defaultCfg.ConnectionNum = cfg.ConnectionNum
	}
	if cfg.Mode != "" {
		defaultCfg.Mode = cfg.Mode
	}
	if cfg.MessageNum > 0 {
		defaultCfg.MessageNum = cfg.MessageNum
	}
	if cfg.MessageSize > 0 {
		defaultCfg.MessageSize = cfg.MessageSize
	}
	if cfg.Interval > 0 {
		defaultCfg.Interval = cfg.Interval
	}
	if cfg.Duration > 0 {
		defaultCfg.Duration = cfg.Duration
	}
	if cfg.ServerIP != "" {
		defaultCfg.ServerIP = cfg.ServerIP
	}
	if cfg.ServerPort > 0 {
		defaultCfg.ServerPort = cfg.ServerPort
	}
	if cfg.Username != "" {
		defaultCfg.Username = cfg.Username
	}
	if cfg.TargetUser != "" {
		defaultCfg.TargetUser = cfg.TargetUser
	}
}

// RunMain executes the performance test
func RunMain() {
	fmt.Printf("Starting performance test in %s mode\n", defaultCfg.Mode)
	fmt.Printf("Connections: %d, Messages per conn: %d, Interval: %dms\n",
		defaultCfg.ConnectionNum, defaultCfg.MessageNum, defaultCfg.Interval)

	startTime := time.Now()

	switch defaultCfg.Mode {
	case ConnectMode:
		testConnections()
	case SendMode:
		testSendMessages()
	case ReceiveMode:
		testReceiveMessages()
	case FullTestMode:
		testFullCycle()
	default:
		fmt.Println("Unknown test mode, using connect mode")
		testConnections()
	}

	result.TotalDuration = time.Since(startTime)
	printResult()
}

// Test connections
func testConnections() {
	var wg sync.WaitGroup
	wg.Add(int(defaultCfg.ConnectionNum))

	for i := range int(defaultCfg.ConnectionNum) {
		go func(idx int) {
			defer wg.Done()
			username := fmt.Sprintf("%s_%d", defaultCfg.Username, idx)

			client := sdk.NewChat(
				net.ParseIP(defaultCfg.ServerIP),
				defaultCfg.ServerPort,
				username,
				"test",
				"test",
			)

			if client == nil {
				atomic.AddInt32(&result.ConnFail, 1)
				fmt.Printf("Connection failed for user %s\n", username)
				return
			}

			atomic.AddInt32(&result.ConnSuccess, 1)
			// Keep the connection open for a while before closing
			time.Sleep(time.Second)
			client.Close()
		}(i)
	}

	wg.Wait()
}

// Test sending upstream messages
func testSendMessages() {
	var wg sync.WaitGroup
	wg.Add(int(defaultCfg.ConnectionNum))

	// Generate test message content
	msgContent := make([]byte, defaultCfg.MessageSize)
	for i := range msgContent {
		msgContent[i] = 'A' + byte(i%26)
	}

	for i := range int(defaultCfg.ConnectionNum) {
		go func(idx int) {
			defer wg.Done()
			username := fmt.Sprintf("%s_%d", defaultCfg.Username, idx)

			client := sdk.NewChat(
				net.ParseIP(defaultCfg.ServerIP),
				defaultCfg.ServerPort,
				username,
				"test",
				"test",
			)

			if client == nil {
				atomic.AddInt32(&result.ConnFail, 1)
				return
			}

			atomic.AddInt32(&result.ConnSuccess, 1)

			// Send messages
			for j := range int(defaultCfg.MessageNum) {
				msg := fmt.Sprintf("%s: Test message %d", string(msgContent[:20]), j)

				// According to SDK implementation, create Message object and call Send method
				message := &sdk.Message{
					Type:       sdk.MsgTypeText,
					Name:       username,
					FormUserID: username,
					ToUserID:   defaultCfg.TargetUser,
					Content:    msg,
				}

				client.Send(message)
				atomic.AddInt32(&result.MsgSent, 1)

				// Wait for the specified interval before sending next message
				time.Sleep(time.Duration(defaultCfg.Interval) * time.Millisecond)
			}

			client.Close()
		}(i)
	}

	wg.Wait()
}

// Test receiving downstream messages
func testReceiveMessages() {
	var wg sync.WaitGroup
	wg.Add(int(defaultCfg.ConnectionNum))

	for i := range int(defaultCfg.ConnectionNum) {
		go func(idx int) {
			defer wg.Done()
			username := fmt.Sprintf("%s_%d", defaultCfg.TargetUser, idx)

			client := sdk.NewChat(
				net.ParseIP(defaultCfg.ServerIP),
				defaultCfg.ServerPort,
				username,
				"test",
				"test",
			)

			if client == nil {
				atomic.AddInt32(&result.ConnFail, 1)
				return
			}

			atomic.AddInt32(&result.ConnSuccess, 1)

			// Get the channel for receiving messages
			recvChan := client.Recv()

			// Start a goroutine to listen for messages
			done := make(chan struct{})
			go func() {
				for msg := range recvChan {
					if msg.Type == sdk.MsgTypeText {
						atomic.AddInt32(&result.MsgReceived, 1)
					}
				}
				close(done)
			}()

			// Wait to receive messages
			time.Sleep(time.Duration(defaultCfg.Duration) * time.Second)
			client.Close()
			<-done // Wait for the receiving goroutine to finish
		}(i)
	}

	wg.Wait()
}

// Complete test cycle
func testFullCycle() {
	// Start receivers first
	fmt.Println("Starting receivers...")
	go func() {
		receiverCfg := defaultCfg
		receiverCfg.ConnectionNum = defaultCfg.ConnectionNum / 5 // Number of receivers is 1/5 of senders
		receiverCfg.Mode = ReceiveMode

		var wg sync.WaitGroup
		wg.Add(int(receiverCfg.ConnectionNum))

		for i := range int(receiverCfg.ConnectionNum) {
			go func(idx int) {
				defer wg.Done()
				username := fmt.Sprintf("%s_%d", defaultCfg.TargetUser, idx)

				client := sdk.NewChat(
					net.ParseIP(defaultCfg.ServerIP),
					defaultCfg.ServerPort,
					username,
					"test",
					"test",
				)

				if client == nil {
					return
				}

				// Get the channel for receiving messages
				recvChan := client.Recv()

				// Start a goroutine to listen for messages
				done := make(chan struct{})
				go func() {
					for msg := range recvChan {
						if msg.Type == sdk.MsgTypeText {
							atomic.AddInt32(&result.MsgReceived, 1)
						}
					}
					close(done)
				}()

				// Wait longer than the send test to ensure all messages are received
				time.Sleep(time.Duration(defaultCfg.Duration+10) * time.Second)
				client.Close()
				<-done // Wait for the receiving goroutine to finish
			}(i)
		}

		wg.Wait()
	}()

	// Give receivers some time to establish connections
	time.Sleep(3 * time.Second)

	// Then start senders
	fmt.Println("Starting senders...")
	testSendMessages()
}

// Print test results
func printResult() {
	fmt.Println("\n--------- Test Result ---------")
	fmt.Printf("Test Mode: %s\n", defaultCfg.Mode)
	fmt.Printf("Total Duration: %.2f seconds\n", result.TotalDuration.Seconds())
	fmt.Printf("Connections: Success=%d, Failed=%d, Total=%d\n",
		result.ConnSuccess, result.ConnFail, defaultCfg.ConnectionNum)

	if defaultCfg.Mode == SendMode || defaultCfg.Mode == FullTestMode {
		fmt.Printf("Messages Sent: %d\n", result.MsgSent)
		if result.MsgSent > 0 {
			fmt.Printf("Send Rate: %.2f msgs/second\n", float64(result.MsgSent)/result.TotalDuration.Seconds())
		}
	}

	if defaultCfg.Mode == ReceiveMode || defaultCfg.Mode == FullTestMode {
		fmt.Printf("Messages Received: %d\n", result.MsgReceived)
		if result.MsgReceived > 0 {
			fmt.Printf("Receive Rate: %.2f msgs/second\n", float64(result.MsgReceived)/result.TotalDuration.Seconds())
		}
	}

	if defaultCfg.Mode == FullTestMode && result.MsgSent > 0 {
		fmt.Printf("Message Delivery Rate: %.2f%%\n", float64(result.MsgReceived)/float64(result.MsgSent)*100)
	}
	fmt.Println("-------------------------------")
}
