package cmd

import (
	"github.com/feichai0017/GoChat/perf"
	"github.com/spf13/cobra"
)

var (
	testMode        string
	connectionNum   int32
	messageNum      int32
	messageSize     int32
	messageInterval int
	testDuration    int
	serverIP        string
	serverPort      int
	targetUser      string
)

func init() {
	perfCmd := &cobra.Command{
		Use:   "perf",
		Short: "Run performance tests for GoChat",
		Long:  `Performance testing tool for GoChat, supports connection, sending, and receiving tests.`,
		Run:   runPerf,
	}

	// add command line arguments
	perfCmd.Flags().StringVarP(&testMode, "mode", "m", "connect", "Test mode: connect, send, receive, or full")
	perfCmd.Flags().Int32VarP(&connectionNum, "connections", "c", 100, "Number of connections to create")
	perfCmd.Flags().Int32VarP(&messageNum, "messages", "n", 10, "Number of messages per connection")
	perfCmd.Flags().Int32VarP(&messageSize, "size", "s", 128, "Message size in bytes")
	perfCmd.Flags().IntVarP(&messageInterval, "interval", "i", 100, "Interval between messages (ms)")
	perfCmd.Flags().IntVarP(&testDuration, "duration", "d", 60, "Test duration in seconds (for receive mode)")
	perfCmd.Flags().StringVar(&serverIP, "ip", "127.0.0.1", "Server IP address")
	perfCmd.Flags().IntVar(&serverPort, "port", 8900, "Server port")
	perfCmd.Flags().StringVarP(&targetUser, "target", "t", "perf_receiver", "Target user for message sending")

	rootCmd.AddCommand(perfCmd)
}

func runPerf(cmd *cobra.Command, args []string) {
	var testModeEnum perf.TestMode

	// convert mode string to enum
	switch testMode {
	case "connect":
		testModeEnum = perf.ConnectMode
	case "send":
		testModeEnum = perf.SendMode
	case "receive":
		testModeEnum = perf.ReceiveMode
	case "full":
		testModeEnum = perf.FullTestMode
	default:
		cmd.Printf("Invalid test mode: %s. Using default: connect\n", testMode)
		testModeEnum = perf.ConnectMode
	}

	// set test config
	cfg := perf.TestConfig{
		Mode:          testModeEnum,
		ConnectionNum: connectionNum,
		MessageNum:    messageNum,
		MessageSize:   messageSize,
		Interval:      messageInterval,
		Duration:      testDuration,
		ServerIP:      serverIP,
		ServerPort:    serverPort,
		Username:      "perf_tester",
		TargetUser:    targetUser,
	}

	// apply config and run test
	perf.SetConfig(cfg)
	perf.RunMain()
}
