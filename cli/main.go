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
	"charm.land/lipgloss/v2"

	tea "charm.land/bubbletea/v2"
	"github.com/nishitjha/drop/discovery"
	"github.com/nishitjha/drop/internal"
	"github.com/nishitjha/drop/internal/archive"
	"github.com/nishitjha/drop/webserver"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
	s.WriteString("\nPick a file or directory for sharing. Use the arrow keys or your mouse wheel to scroll and navigate. \nPress enter to choose the selected file.\n\n")

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
	Use:   "drop",
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
	Use:     "list",
	Aliases: []string{"ls", "devices", "peers"},
	Short:   "Use drop [list/ls/devices/peers] to list all devices with Drop on this network.",
	Run: func(cmd *cobra.Command, args []string) {
		var devices map[string]discovery.Device
		internal.RunSpinner("Scanning for devices...", func() tea.Msg {
			discovery.ServiceBrowser()
			time.Sleep(2 * time.Second)
			devices = discovery.Devices.List()
			return internal.TaskResultMsg{}
		})

		if len(devices) == 0 {
			fmt.Printf("%s Couldn't find any devices on your network. Make sure they're running Drop and try again.\n", internal.Icons.Negative)
			return
		}

		// time for some math oogabooga to figure out the widths of columns
		maxName, maxStatus, maxIP := 11, 9, 10

		for _, d := range devices {
			if len(d.DeviceName) > maxName {
				maxName = len(d.DeviceName)
			}
			if len(d.Address) > maxIP {
				maxIP = len(d.Address)
			}
		}

		nameCol := lipgloss.NewStyle().Width(maxName + 4)
		statusCol := lipgloss.NewStyle().Width(maxStatus + 4)
		ipCol := lipgloss.NewStyle().Width(maxIP + 4)

		fmt.Println()
		fmt.Print(nameCol.Bold(true).Render("Device Name"))
		fmt.Print(statusCol.Bold(true).Render("Status"))
		fmt.Print(ipCol.Bold(true).Render("IP Address"))
		fmt.Println()

		dividerWidth := maxName + maxStatus + maxIP + 12
		fmt.Println(lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(strings.Repeat("─", dividerWidth)))

		for _, d := range devices {
			status := lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Render("Available")
			if d.Status == 1 {
				status = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("Busy")
			}

			fmt.Print(nameCol.Render(d.DeviceName))
			fmt.Print(statusCol.Render(status))
			fmt.Print(ipCol.Render(d.Address))
			fmt.Println()
		}
		fmt.Println()
	},
}

var textMode bool

var share = &cobra.Command{
	Use:     "share deviceName [file_path]",
	Aliases: []string{"sh", "send"},
	Short:   "Use drop [share/sh/send] {device_name} {file_path} to attempt streaming a file across to said device.",
	Long:  "Use drop [share/sh/send] {device_name} {file_path} to attempt streaming a file across to said device. \n Use the -t or --text flag to share a text snippet instead of a file. \n Example: drop share -t {device_name} \"Hello, world!\"",
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

		textMode, _ = cmd.Flags().GetBool("text")


		if textMode {
			var textSnippet string
			if len(args) > 1 {
				textSnippet = args[1]
			} else {
				fmt.Printf("%s You forgot to pass in the text! Use \"drop share --t {device_name} {text_snippet}\" to share said text snippet.\n", internal.Icons.Information)
				return
			}
			fmt.Println(textSnippet)
			result := internal.RunSpinner(fmt.Sprintf("Sent a text share request to \"%s\". The device has 3 minutes to accept it.", targetDevice.DeviceName), func() tea.Msg {
			httpClient := &http.Client{}

			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
			defer cancel()

			reqURL := fmt.Sprintf("http://%s:3000/request?senderName=%s&t=%v&UUID=%s", targetDevice.Address, discovery.InstanceName, true, targetDevice.UUID)
		
		
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
			fmt.Printf("%s Great success! \"%s\" accepted your text sharing request.\n", internal.Icons.Positive, targetDevice.DeviceName)

			internal.Launch(targetDevice.Address, targetDevice.DeviceName, "", textSnippet)
		} else if result.Response.StatusCode == http.StatusForbidden || result.Response.StatusCode == http.StatusUnauthorized {
			fmt.Printf("%s What a fucking loser. \"%s\" declined your text sharing request.\n", internal.Icons.Negative, targetDevice.DeviceName)
		}
		
			return
		}
	

		var fileInfo os.FileInfo
		var err error
		if (len(args) > 1) {
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


		result := internal.RunSpinner(fmt.Sprintf("Sent a share request to \"%s\". The device has 3 minutes to accept it.", targetDevice.DeviceName), func() tea.Msg {
			httpClient := &http.Client{}

			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
			defer cancel()

			reqURL := fmt.Sprintf("http://%s:3000/request?senderName=%s&t=%v&UUID=%s&fileName=%s&fileSize=%d", targetDevice.Address, discovery.InstanceName, false, targetDevice.UUID, func() string {
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
				info, err := os.Stat(args[1])
				if err != nil {
					fmt.Printf("%s Error opening file/directory \"%s\": %v\n", internal.Icons.Negative, args[1], err)
					return
				}

				if info.IsDir() {
					archive.ArchiveDirectoryToZip(args[1])
					return
				}

				internal.Launch(targetDevice.Address, targetDevice.DeviceName, args[1], "")
				return
			}

			picker := filepicker.New()
			picker.DirAllowed = true
			picker.FileAllowed = true

			homeDir, _ := os.UserHomeDir()
			picker.CurrentDirectory = homeDir
			pickerModel := model{filepicker: picker}
			p := tea.NewProgram(pickerModel)
			finalModel, err := p.Run()


			if err != nil {
				fmt.Printf("%s Error running the file/directory picker: %v\n", internal.Icons.Negative, err)
				fmt.Printf("%s Maybe you'll have some luck passing in the file/directory path in the command itself? Use \"drop share --help\" to see how to do that.\n", internal.Icons.Information)
				return
			}

			selectedModel := finalModel.(model)
			if selectedModel.selectedFile == "" {
				fmt.Println("No file/directory selected for sharing. Exiting Drop.")
				return
			}

			info, err := os.Stat(selectedModel.selectedFile)
			if err != nil {
				fmt.Printf("%s Error opening file/directory \"%s\": %v\n", internal.Icons.Negative, selectedModel.selectedFile, err)
				return
			}

			if info.IsDir() {
				archive.ArchiveDirectoryToZip(selectedModel.selectedFile)
				return
			}

			internal.Launch(targetDevice.Address, targetDevice.DeviceName, selectedModel.selectedFile, "")

		} else if result.Response.StatusCode == http.StatusForbidden || result.Response.StatusCode == http.StatusUnauthorized {
			fmt.Printf("%s What a fucking loser. \"%s\" declined your sharing request.\n", internal.Icons.Negative, targetDevice.DeviceName)
		}
	},
}

var config = &cobra.Command{
	Use:     "config [setting] [newValue]",
	Aliases: []string{"settings", "con", "conf"},
	Short:   "Use drop [config/settings/con/conf] to view Drop's configuration. Use drop [config/settings/con] {setting} {newValue} to change a setting.",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			for _, key := range viper.AllKeys() {
				fmt.Printf("%s: %v\n", key, viper.Get(key))
			}
			return
		}

		if len(args) == 1 {
			fmt.Printf("%s: %v\n", args[0], viper.Get(args[0]))
			return
		}
	},
}

func init() {
	rootCmd.AddCommand(list, share, config)
	share.Flags().BoolVarP(&textMode, "text", "t", false, "Share a text snippet instead of a file.")
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		fmt.Printf("%s Error: %v\n", internal.Icons.Negative, err)
	}
}
