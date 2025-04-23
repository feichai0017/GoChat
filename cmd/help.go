package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(helpCmd)
}

var helpCmd = &cobra.Command{
	Use: "help",
	Run: HelpHandle,
}

func HelpHandle(cmd *cobra.Command, args []string) {
	fmt.Println("GoChat Interactive Shell")
	fmt.Println("This is a Amazing IM system.")
	fmt.Println("")
	fmt.Println("Available Commands:")
	fmt.Println("  client     Start the client chat window")
	fmt.Println("  help       Display this help information")
	fmt.Println("  exit       Exit the shell")
	fmt.Println("")
}
