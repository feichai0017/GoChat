package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gookit/color"
	"github.com/spf13/cobra"
)

func init() {
	cobra.OnInitialize(initConfig)
}

var rootCmd = &cobra.Command{
	Use:   "gochat",
	Short: "Amazing IM System",
}

func Execute() {
	// if no arguments were provided, run the interactive shell directly.
	if len(os.Args) <= 1 {
		GoChat(nil, nil)
		return // Exit after the shell finishes
	}

	// Arguments were provided, let Cobra handle command routing.
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func GoChat(cmd *cobra.Command, args []string) {
	// Enhanced ASCII Art and Info
	color.Cyan.Println(`
  ██████╗  ██████╗  ██████╗ ██╗  ██╗ █████╗ ████████╗
 ██╔════╝ ██║   ██╗██╔════╝ ██║  ██║██╔══██╗╚══██╔══╝
 ██║  ███╗██║   ██║██║      ███████║███████║   ██║
 ██║   ██║██║   ██║██║      ██╔══██║██╔══██║   ██║
 ╚██████╔╝╚██████╔╝╚██████╗ ██║  ██║██║  ██║   ██║
  ╚═════╝  ╚═════╝  ╚═════╝ ╚═╝  ╚═╝╚═╝  ╚═╝   ╚═╝
	`)

	version := "0.0.1"
	pid := os.Getpid()

	color.New(color.FgLightWhite, color.BgBlue).Println(" GoChat Interactive Shell v" + version + " ")
	color.FgLightCyan.Printf(" PID: %d | Started: %s | Type 'help' or 'exit' ", pid, time.Now().Format("15:04:05"))
	fmt.Println() // Add a newline for spacing

	// --- Interactive Shell Logic ---
	reader := bufio.NewReader(os.Stdin)
	for {
		color.New(color.FgGreen, color.OpBold).Print("gochat > ")
		input, err := reader.ReadString('\n')
		if err != nil {
			// Handle EOF (Ctrl+D) or other errors gracefully
			fmt.Println() // Newline after Ctrl+D
			color.FgRed.Println("Exiting GoChat shell due to input error or EOF.")
			break
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue // Skip empty input
		}

		// Handle exit command
		if input == "exit" || input == "quit" {
			color.FgRed.Println("Exiting GoChat shell.")
			break
		}

		// Parse the input into command and arguments
		parts := strings.Fields(input)

		// Find the command in Cobra using the root command
		foundCmd, remainingArgs, err := rootCmd.Find(parts)
		if err != nil || foundCmd == nil {
			color.FgRed.Printf("Error: Unknown command '%s'. Type 'help' for available commands.\n", parts[0])
			continue
		}

		// Prevent recursive call to the shell itself
		if foundCmd == rootCmd {
			color.FgYellow.Println("You are already in the GoChat shell. Type 'help' or a command name.")
			continue
		}

		// Execute the found command
		foundCmd.SetArgs(remainingArgs) // Set args for context

		var execErr error
		if foundCmd.RunE != nil {
			// Prefer RunE if available for other commands
			execErr = foundCmd.RunE(foundCmd, remainingArgs)
		} else if foundCmd.Run != nil {
			// Fallback to Run
			foundCmd.Run(foundCmd, remainingArgs)
		} else {
			// If no Run/RunE (and not help), show its help.
			foundCmd.Help()
		}

		if execErr != nil {
			// Handle errors from Execute() or RunE()
			color.FgRed.Printf("Error executing command '%s': %v\n", foundCmd.Name(), execErr)
		}

		// IMPORTANT: Reset arguments on rootCmd for the next Find operation in the loop.
		rootCmd.SetArgs([]string{})
	}
}

func initConfig() {

}
