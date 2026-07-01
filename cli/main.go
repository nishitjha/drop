package cli

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/briandowns/spinner"
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
		sp := spinner.New(spinner.CharSets[40], 100*time.Millisecond)
		sp.Suffix = " Scanning..."
		sp.Start()
		discovery.ServiceBrowser()
		time.Sleep(2 * time.Second)
		sp.Stop()
		devices := discovery.Devices.List()
		if len(devices) == 0 {
			fmt.Println("Couldn't find any devices on your network. Make sure they're running Drop and try again.")
			return
		}
		for _, device := range devices {
			fmt.Printf("Device: %s, IP: %s, Status: %d, Last Updated: %d, UUID: %s\n", device.DeviceName, device.Address, device.Status, device.LastUpdated, device.UUID)
		}
	},
}

var share = &cobra.Command{
	Use: "share deviceName [file_path]",
	Aliases: []string{"share", "sh", "send"},
	Short: "Use drop [share/sh/send] {device_name} {file_path} to attempt streaming a file across to said device.",
	Run: func(cmd *cobra.Command, args []string) {
		discovery.ServiceBrowser()
		devices := discovery.Devices.List()
		
		time.Sleep(2 * time.Second)
		for _, device := range devices {
			if device.DeviceName == args[0] {
				sp := spinner.New(spinner.CharSets[40], 100*time.Millisecond)
				sp.Suffix = fmt.Sprintf(" Sending a share request to \"%[1]s\". The device has 3 minutes to accept it.", device.DeviceName)
				sp.Start()
				
				httpClient := &http.Client{}
				
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
				defer cancel()

				req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("http://%[1]s:3000/request?senderName=%[2]s&UUID=%[3]s", device.Address, discovery.InstanceName, device.UUID), nil)
				if err != nil {
					fmt.Println(err)
					return
				}

				response, err := httpClient.Do(req)
				if err != nil {
					fmt.Printf("Request failed or timed out: %v\n", err)
					return
				}

				sp.Stop()
				
				
				defer response.Body.Close()
				if response.StatusCode == http.StatusOK {
					fmt.Printf("Great success! \"%s\" accepted your sharing request.\n", device.DeviceName)
				} else if response.StatusCode == http.StatusForbidden || response.StatusCode == http.StatusUnauthorized {
					fmt.Printf("What a fucking loser. \"%s\" declined your sharing request.\n", device.DeviceName)
				}
			}
		
		fmt.Printf("Couldn't find \"%[1]s\" on your network. Make sure it's running Drop and try again.", args[0])
		}
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

