package cli

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/professional-procrastinator/drop/discovery"
	"github.com/professional-procrastinator/drop/webserver"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use: "drop",
	Short: "Start the Drop discovery and broadcast daemon.",
	Run: func(cmd *cobra.Command, args []string) {
		go discovery.LaunchService()
		discovery.ServiceBrowser() //runs continously in the background already

		go webserver.Listen()

		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
		
		<-sig 
		fmt.Println("Au revoir!")
		os.Exit(0)
	},
}

var list = &cobra.Command{
	Use: "list",
	Aliases: []string{"ls", "devices", "peers"},
	Short: "Use drop [list/ls/devices/peers] to list all devices with Drop on this network.",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("no devices for your lonely ass :rofl:")
	},
}

var share = &cobra.Command{
	Use: "share",
	Aliases: []string{"share", "sh", "send"},
	Short: "Use drop [share/sh/send] {device_name} to attempt streaming a file across to said device.",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("no devices for your lonely ass :rofl:")
	},
}

func init(){
	rootCmd.AddCommand(list, share)
}

func Execute() {
	err := rootCmd.Execute() 
	if err != nil {
		panic(err)
	}
}