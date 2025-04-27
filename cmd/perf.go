package cmd

import (
	"github.com/feichai0017/GoChat/perf"
	"github.com/spf13/cobra"
)


func init() {
	rootCmd.AddCommand(perfCmd)
	perfCmd.PersistentFlags().Int32Var(&perf.TcpConnNum, "tcp_conn_num", 10000, "tcp connection number, default 10000")
}

var perfCmd = &cobra.Command{
	Use: "perf",
	Run: PerfHandle,
}

func PerfHandle(cmd *cobra.Command, args []string) {
	perf.RunMain()
}
