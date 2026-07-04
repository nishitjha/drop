package cli

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"charm.land/bubbles/v2/filepicker"
	"charm.land/bubbles/v2/spinner"
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
		return tea.NewView("") // cleaer the terminal on quitting
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

var _ tea.Model = spinnerModel{}

func (m spinnerModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.task)
}

func (m spinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg: 
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

func (m spinnerModel) View() tea.View {
	str := fmt.Sprintf("\n %s %s\n", m.spinner.View(), m.text)
	return tea.NewView(str)
}

func runSpinner(text string, task func() tea.Msg) taskResultMsg {
	s := spinner.New()
	s.Spinner = spinner.Monkey

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

		if len(args) == 0 {
			fmt.Println("You forgot to specify a device! Use \"drop ls\" to see a list of available devices.")
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
					
					picker := filepicker.New()

					homeDir, _ := os.UserHomeDir()
					picker.CurrentDirectory = homeDir
					pickerModel := model{filepicker: picker}
					p := tea.NewProgram(pickerModel)
					finalModel, err := p.Run()
					if err != nil {
						fmt.Printf("Error running the file picker: %v\n", err)
						fmt.Printf("Maybe you'll have some luck passing in the file path in the command itself? Use \"drop share --help\" to see how to do that.")
						return
					}

					selectedModel := finalModel.(model)
					if selectedModel.selectedFile == "" {
						fmt.Println("No file selected for sharing. Exiting Drop.")
						return
					}
					
					// stream the file to the device
					internal.StreamFile(device.Address, device.UUID, selectedModel.selectedFile)
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

