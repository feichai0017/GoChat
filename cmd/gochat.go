package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/gookit/color"
	"github.com/spf13/cobra"
)

var (
	ConfigPath string
)

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&ConfigPath, "config", "./gochat.yaml", "config file (default is ./gochat.yaml)")

	// Set PersistentPreRun to print logo before any command
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		printLogo()
	}
}

var rootCmd = &cobra.Command{
	Use:   "gochat",
	Short: "Amazing IM System",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// printLogo prints the GoChat logo and basic info
func printLogo() {
	color.Cyan.Println(`
  ██████╗  ██████╗  ██████╗ ██╗  ██╗ █████╗ ████████╗
 ██╔════╝ ██╔═══██╗██╔════╝ ██║  ██║██╔══██╗╚══██╔══╝
 ██║  ███╗██║   ██║██║      ███████║███████║   ██║
 ██║   ██║██║   ██║██║      ██╔══██║██╔══██║   ██║
 ╚██████╔╝╚██████╔╝╚██████╗ ██║  ██║██║  ██║   ██║
  ╚═════╝  ╚═════╝  ╚═════╝ ╚═╝  ╚═╝╚═╝  ╚═╝   ╚═╝
	`)

	version := "0.0.1"
	pid := os.Getpid()

	color.New(color.FgLightWhite, color.BgBlue).Println(" GoChat v" + version + " ")
	color.FgLightCyan.Printf(" PID: %d | Started: %s ", pid, time.Now().Format("15:04:05"))
	fmt.Println() 
}

func initConfig() {
	// Configuration initialization code
}
