package cli

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nishitjha/drop/discovery"
	"github.com/nishitjha/drop/webserver"
	"github.com/spf13/cobra"
)

type taskResultMsg struct {
	response *http.Response
	err      error
}

type spinnerModel struct {
	spinner spinner.Model
	text    string
	task    func() tea.Msg
	result  taskResultMsg
}

func (m spinnerModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.task)
}

func (m spinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			os.Exit(0)
		}
	case taskResultMsg:
		m.result = msg
		return m, tea.Quit
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m spinnerModel) View() string {
	return fmt.Sprintf("\n %s %s\n", m.spinner.View(), m.text)
}

func runSpinner(text string, task func() tea.Msg) taskResultMsg {
	s := spinner.New()
	s.Spinner = spinner.Monkey
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	m := spinnerModel{
		spinner: s,
		text:    text,
		task:    task,
	}

	p := tea.NewProgram(m)
	finalModel, _ := p.Run()
	return finalModel.(spinnerModel).result
}

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
		runSpinner("Scanning for devices...", func() tea.Msg {
			discovery.ServiceBrowser()
			time.Sleep(2 * time.Second)
			return taskResultMsg{}
		})
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

		if (len(devices) == 0) {
			fmt.Println("Couldn't find any devices on your network. Make sure they're running Drop and try again.")
			return
		}

		for _, device := range devices {
			if device.DeviceName == args[0] {
				result := runSpinner(fmt.Sprintf("Sent a share request to \"%s\". The device has 3 minutes to accept it.", device.DeviceName), func() tea.Msg {
			httpClient := &http.Client{}

			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
			defer cancel()

			reqURL := fmt.Sprintf("http://%s:3000/request?senderName=%s&UUID=%s", device.Address, discovery.InstanceName, device.UUID)
			req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
			if err != nil {
				return taskResultMsg{err: err}
			}

			response, err := httpClient.Do(req)
			return taskResultMsg{response: response, err: err}
		})
				if result.err != nil {
					fmt.Println("The request timed out. Maybe they missed it? (either that or they hate you).")
					return
				}

				
				defer result.response.Body.Close()
				if result.response.StatusCode == http.StatusOK {
					fmt.Printf("Great success! \"%s\" accepted your sharing request.\n", device.DeviceName)
				} else if result.response.StatusCode == http.StatusForbidden || result.response.StatusCode == http.StatusUnauthorized {
					fmt.Printf("What a fucking loser. \"%s\" declined your sharing request.\n", device.DeviceName)
				}
			} else {
				
				fmt.Printf("Couldn't find \"%[1]s\" on your network. Make sure it's running Drop and try again.", args[0])
		}}
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

