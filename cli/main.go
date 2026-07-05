package cli

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"charm.land/bubbles/v2/filepicker"
	tea "charm.land/bubbletea/v2"
	"github.com/nishitjha/drop/discovery"
	"github.com/nishitjha/drop/internal"
	"github.com/nishitjha/drop/webserver"
	"github.com/spf13/cobra"
)

type model struct {
	filepicker   filepicker.Model
	selectedFile string
	quitting     bool
	err          error
}

type clearErrorMsg struct{}

func clearErrorAfter(t time.Duration) tea.Cmd {
	return tea.Tick(t, func(_ time.Time) tea.Msg {
		return clearErrorMsg{}
	})
}

func (m model) Init() tea.Cmd {
	return m.filepicker.Init()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		}
	case clearErrorMsg:
		m.err = nil
	}

	var cmd tea.Cmd
	m.filepicker, cmd = m.filepicker.Update(msg)

	if didSelect, path := m.filepicker.DidSelectFile(msg); didSelect {
		m.selectedFile = path
		return m, tea.Quit
	}

	return m, cmd
}

func (m model) View() tea.View {
	if m.quitting {
		return tea.NewView("")
	}
	
	var s strings.Builder
	s.WriteString("\nPick a file for sharing. Use the arrow keys or your mouse wheel to scroll and navigate. \nPress enter to choose the selected file.\n\n")
	
	if m.err != nil {
		s.WriteString(m.filepicker.Styles.DisabledFile.Render(m.err.Error()))
	} else if m.selectedFile == "" {
		s.WriteString("Pick a file:")
	} else {
		s.WriteString("Selected file: " + m.filepicker.Styles.Selected.Render(m.selectedFile))
	}
	
	s.WriteString("\n\n" + m.filepicker.View() + "\n")
	
	v := tea.NewView(s.String())
	v.AltScreen = true
	return v
}

var rootCmd = &cobra.Command{
	Use: "drop",
	Short: "Start the Drop discovery and broadcast daemon.",
	Run: func(cmd *cobra.Command, args []string) {
		go discovery.LaunchService()
		discovery.ServiceBrowser()

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
		internal.RunSpinner("Scanning for devices...", func() tea.Msg {
			discovery.ServiceBrowser()
			time.Sleep(2 * time.Second)
			return internal.TaskResultMsg{}
		})
		devices := discovery.Devices.List()
		if len(devices) == 0 {
			fmt.Printf("%s Couldn't find any devices on your network. Make sure they're running Drop and try again.\n", internal.Icons.Negative)
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
		if len(args) == 0 {
			fmt.Printf("%s You forgot to specify a device! Use \"drop ls\" to see a list of devices available for sharing.\n", internal.Icons.Information)
			return
		}

		discovery.ServiceBrowser()
		devices := discovery.Devices.List()
		
		time.Sleep(2 * time.Second)

		if len(devices) == 0 {
			fmt.Printf("%s Couldn't find any devices on your network. Make sure they're running Drop and try again.\n", internal.Icons.Negative)
			return
		}

		var fileInfo os.FileInfo
		var err error
		if len(args) > 1 {
			if _, err := os.Stat(args[1]); os.IsNotExist(err) {
				fmt.Printf("%s The file \"%s\" does not exist. Make sure you typed the absolute/relative path correctly and try again.\n", internal.Icons.Negative, args[1])
				return
			}
			
			fileInfo, err = os.Stat(args[1])
			if err != nil {
				fmt.Printf("%s Error opening file \"%s\": %v\n", internal.Icons.Negative, args[1], err)
				return
			}
		}

		var targetDevice *discovery.Device
		for _, device := range devices {
			if device.DeviceName == args[0] {
				d := device
				targetDevice = &d
				break
			}
		}

		if targetDevice == nil {
			fmt.Printf("%s Couldn't find \"%s\" on your network. Make sure it's running Drop and try again.\n", internal.Icons.Negative, args[0])
			return
		}

		result := internal.RunSpinner(fmt.Sprintf("Sent a share request to \"%s\". The device has 3 minutes to accept it.", targetDevice.DeviceName), func() tea.Msg {
			httpClient := &http.Client{}

			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
			defer cancel()

			reqURL := fmt.Sprintf("http://%s:3000/request?senderName=%s&UUID=%s&fileName=%s&fileSize=%d", targetDevice.Address, discovery.InstanceName, targetDevice.UUID, func() string {
				if len(args) > 1 {
					return fileInfo.Name()
				}
				return ""
			}(), func() int64 {
				if len(args) > 1 {
					return fileInfo.Size()
				}
				return 0
			}())
			
			req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
			if err != nil {
				return internal.TaskResultMsg{Error: err}
			}

			response, err := httpClient.Do(req)
			return internal.TaskResultMsg{Response: response, Error: err}
		})

		if result.Error != nil {
			fmt.Printf("%s The request timed out. Maybe they missed it? (either that or they hate you).\n", internal.Icons.Information)
			return
		}
		
		defer result.Response.Body.Close()
		
		if result.Response.StatusCode == http.StatusOK {
			fmt.Printf("%s Great success! \"%s\" accepted your sharing request.\n", internal.Icons.Positive, targetDevice.DeviceName)

			if len(args) > 1 {
				streamResult := internal.RunSpinner(fmt.Sprintf("Streaming \"%s\" to \"%s\"...", filepath.Base(args[1]), targetDevice.DeviceName), func() tea.Msg {
					err := internal.StreamFile(targetDevice.Address, targetDevice.DeviceName, args[1])
					return internal.TaskResultMsg{Error: err}
				})
				
				if streamResult.Error != nil {
	// Added \r\033[2K to the start!
	fmt.Printf("\r\033[2K%s Error streaming file: %v\n", internal.Icons.Negative, streamResult.Error)
} else {
	// Added \r\033[2K to the start!
	fmt.Printf("\r\033[2K%s The file \"%s\" has been sent successfully to %s.\n", internal.Icons.Positive, filepath.Base(args[1]), targetDevice.DeviceName)
}
				return
			}

			picker := filepicker.New()

			homeDir, _ := os.UserHomeDir()
			picker.CurrentDirectory = homeDir
			pickerModel := model{filepicker: picker}
			p := tea.NewProgram(pickerModel)
			finalModel, err := p.Run()
			if err != nil {
				fmt.Printf("%s Error running the file picker: %v\n", internal.Icons.Negative, err)
				fmt.Printf("%s Maybe you'll have some luck passing in the file path in the command itself? Use \"drop share --help\" to see how to do that.\n", internal.Icons.Information)
				return
			}

			selectedModel := finalModel.(model)
			if selectedModel.selectedFile == "" {
				fmt.Println("No file selected for sharing. Exiting Drop.")
				return
			}

			streamResult := internal.RunSpinner(fmt.Sprintf("Streaming \"%s\" to \"%s\"...", filepath.Base(selectedModel.selectedFile), targetDevice.DeviceName), func() tea.Msg {
				err := internal.StreamFile(targetDevice.Address, targetDevice.DeviceName, selectedModel.selectedFile)
				return internal.TaskResultMsg{Error: err}
			})

			if streamResult.Error != nil {
	fmt.Printf("\r\033[2K%s Error streaming file: %v\n", internal.Icons.Negative, streamResult.Error)
} else {
	fmt.Printf("\r\033[2K%s The file \"%s\" has been sent successfully to %s.\n", internal.Icons.Positive, filepath.Base(selectedModel.selectedFile), targetDevice.DeviceName)
}
			
		} else if result.Response.StatusCode == http.StatusForbidden || result.Response.StatusCode == http.StatusUnauthorized {
			fmt.Printf("%s What a fucking loser. \"%s\" declined your sharing request.\n", internal.Icons.Negative, targetDevice.DeviceName)
		}
	},
}

func init(){
	rootCmd.AddCommand(list, share)
}

func Execute() {
	err := rootCmd.Execute() 
	if err != nil {
		fmt.Printf("%s Error: %v\n", internal.Icons.Negative, err)
	}
}