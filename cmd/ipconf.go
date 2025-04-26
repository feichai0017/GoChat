package cmd

import (

	"github.com/spf13/cobra"
	"github.com/feichai0017/GoChat/ipconf"
)

func init() {
	rootCmd.AddCommand(ipConfCmd)
}

var ipConfCmd = &cobra.Command{
	Use: "ipconf",
	Run: IpConfHandle,
}

func IpConfHandle(cmd *cobra.Command, args []string) {
	ipconf.RunMain(ConfigPath)
}





