package cli

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"slices"
	"syscall"
	"time"

	"charm.land/bubbles/v2/filepicker"
	"charm.land/lipgloss/v2"

	tea "charm.land/bubbletea/v2"
	"github.com/nishitjha/drop/daemon"
	"github.com/nishitjha/drop/discovery"
	"github.com/nishitjha/drop/internal"
	"github.com/nishitjha/drop/internal/archive"
	"github.com/nishitjha/drop/webserver"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:   "drop",
	Short: "Start the Drop discovery and broadcast daemon.",
	Run: func(cmd *cobra.Command, args []string) {
		server := discovery.LaunchService()
		discovery.ServiceBrowser()

		go webserver.Listen("user")

		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

		<-sig
		fmt.Printf("%s Au revoir!", internal.Icons.Bye)

		if server != nil {
			server.Shutdown()
		}
		time.Sleep(200 * time.Millisecond)

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

		headers := []string{"Device Name", "Status", "IP Address"}
		var rows [][]string

		for _, d := range devices {
			status := lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Render("Available")
			if d.Status == 1 {
				status = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("Busy")
			}

			rows = append(rows, []string{
				d.DeviceName,
				status,
				d.Address,
			})
		}

		internal.PrintTable(headers, rows, false)
	},
}

var textMode bool
var dirMode bool

var share = &cobra.Command{
	Use:     "share deviceName [file_path]",
	Aliases: []string{"sh", "send"},
	Short:   "Use drop [share/sh/send] {device_name} {file_path} to attempt streaming a file across to said device.",
	Long:    "Use drop [share/sh/send] {device_name} {file_path} to attempt streaming a file across to said device. \n Use the -t or --text flag to share a text snippet instead of a file. \n Example: drop share -t {device_name} \"Hello, world!\"",
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
		dirMode, _ = cmd.Flags().GetBool("dir")

		if textMode {
			if dirMode {
				fmt.Printf("%s You cannot use both the --text and --dir flags at the same time.\n", internal.Icons.Negative)
				return
			}
			var textSnippet string
			if len(args) > 1 {
				textSnippet = args[1]
			} else {
				fmt.Printf("%s You forgot to pass in the text! Use \"drop share --t {device_name} {text_snippet}\" to share said text snippet.\n", internal.Icons.Information)
				return
			}
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
		if len(args) > 1 {
			if _, err := os.Stat(args[1]); os.IsNotExist(err) {
				fmt.Printf("%s The file or directory \"%s\" does not exist. Make sure you typed the absolute/relative path correctly and try again.\n", internal.Icons.Negative, args[1])
				return
			}

			fileInfo, err = os.Stat(args[1])
			if err != nil {
				fmt.Printf("%s Error opening file or directory \"%s\": %v\n", internal.Icons.Negative, args[1], err)
				return
			}

			if fileInfo.IsDir() && !dirMode {
				fmt.Printf("%s The path specified points to a directory. If you intend to share a folder, you must use the --dir/d flag.\n", internal.Icons.Information)
				return
			}

			if !fileInfo.IsDir() && dirMode {
				fmt.Printf("%s The path specified points to a file. If you intend to share a file, you must not use the --dir/d flag.\n", internal.Icons.Information)
				return
			}
		}

		result := internal.RunSpinner(fmt.Sprintf("Sent a share request to \"%s\". The device has 3 minutes to accept it.", targetDevice.DeviceName), func() tea.Msg {
			httpClient := &http.Client{}

			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
			defer cancel()

			reqURL := fmt.Sprintf("http://%s:3000/request?senderName=%s&t=%v&d=%v&UUID=%s&fileName=%s&fileSize=%d", targetDevice.Address, discovery.InstanceName, false, dirMode, targetDevice.UUID, func() string {
				if len(args) > 1 {
					return fileInfo.Name()
				}
				return ""
			}(), func() int64 {
				if len(args) > 1 {
					if fileInfo.IsDir() {
						return 0
					}
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
					fmt.Printf("%s Error opening the file or directory \"%s\": %v\n", internal.Icons.Negative, args[1], err)
					return
				}

				if info.IsDir() {
					archive.Execute(args[1], targetDevice.Address, targetDevice.DeviceName)
					return
				}

				internal.Launch(targetDevice.Address, targetDevice.DeviceName, args[1], "")
				return
			}

			picker := filepicker.New()
			picker.DirAllowed = dirMode
			picker.FileAllowed = !dirMode

			homeDir, _ := os.UserHomeDir()
			picker.CurrentDirectory = homeDir
			pickerModel := internal.FileModel{Filepicker: picker, DirMode: dirMode}
			p := tea.NewProgram(pickerModel)
			finalModel, err := p.Run()

			if err != nil {
				fmt.Printf("%s Error running the %s picker: %v\n", internal.Icons.Negative, func() string {
					if dirMode {
						return "directory"
					}
					return "file"
				}(), err)
				fmt.Printf("%s Maybe you'll have some luck passing in the %s path in the command itself? Use \"drop share --help\" to see how to do that.\n", internal.Icons.Information, func() string {
					if dirMode {
						return "directory"
					}
					return "file"
				}())
				return
			}

			selectedModel := finalModel.(internal.FileModel)
			if selectedModel.SelectedFile == "" {
				fmt.Printf("No %s selected for sharing. Exiting Drop.\n", func() string {
					if dirMode {
						return "directory"
					}
					return "file"
				}())
				return
			}

			info, err := os.Stat(selectedModel.SelectedFile)
			if err != nil {
				fmt.Printf("%s Error opening the %s \"%s\": %v\n", internal.Icons.Negative, func() string {
					if dirMode {
						return "directory"
					}
					return "file"
				}(), selectedModel.SelectedFile, err)
				return
			}

			if info.IsDir() {
				archive.Execute(selectedModel.SelectedFile, targetDevice.Address, targetDevice.DeviceName)
				return
			}

			internal.Launch(targetDevice.Address, targetDevice.DeviceName, selectedModel.SelectedFile, "")

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
			headers := []string{"Setting", "Current Value", "Description"}
			var rows [][]string

			var description = map[string]string{
				"sharing.receiveDir":                 "Default folder where incoming files are saved.",
				"sharing.isDiscoverable":             "Choose whether this device is visible to others on the network via mDNS.",
				"sharing.askReceiveDirEverytime":     "Choose whether you should be asked where to save incoming files everytime instead of using the default.",
				"sharing.trustAllDevices":            "Skip confirmation prompts and accept requests from all devices automatically.",
				"sharing.trustedDevices":             "Set of devices allowed to send files without you having to accept a sharing request.",
				"sharing.autoRejectUntrustedDevices": "Automatically reject incoming requests from devices not in the trusted list.",
				"sharing.autoRenameExistingFiles":    "Append a suffix to incoming files instead of overwriting existing ones with the same name.",

				"sharing.acceptTextSnippetsByDefault": "Automatically accept incoming plain-text/clipboard snippets without a prompt.",
				"sharing.autoCopyToClipboard":         "Copy received text snippets to the clipboard automatically.",

				"sharing.advanced.enableTransferLog": "Keep a persistent log of completed and failed transfers.",
				"sharing.advanced.logFilePath":       "Path to the transfer history log file.",

				"sharing.folders.archiveFormat":        "Archive format used when sending folders (zip or tar.gz), OS-dependent by default.",
				"sharing.folders.compressionLevel":     "Compression level for folder archives, from 0 (none) to 3 (max); higher levels use more CPU in exchange for smaller transfers.",
				"sharing.folders.intelligentArchive":   "Skip compressing files that don't benefit from it (media, existing archives) and exclude common dev/VCS directories when archiving folders.",
				"sharing.folders.autoExtractOnReceive": "Automatically extract received folder archives instead of leaving them compressed.",

				"webserver.port": "Port the local HTTP server listens on for incoming transfers and the web UI.",

				"discovery.instanceName":         "Name this device advertises to others on the network.",
				"discovery.advanced.serviceName": "mDNS service type used for discovery (advanced, don't change unless you know why).",
				"discovery.advanced.domain":      "mDNS domain used for discovery (advanced, don't change unless you know why).",
				"discovery.advanced.metadata":    "Raw TXT records advertised alongside the mDNS service.",
				"discovery.advanced.port":        "Port used for the mDNS discovery service (separate from the webserver port).",
				"discovery.advanced.deviceUUID":  "Unique identifier for this device, used to distinguish it from others on the network. Do not change this even if you know what you're doing.",
				"network.maxBandwidthMBps":       "Caps outgoing transfer speed in MB/s; 0 here corresponds to no upper bound on the speed.",
			}

			// in case you get a bug here it prolly has to do with the fact that viper.AllKeys() returns the keyname in lowercase
			// the description map has the keys in camelCase so yeah

			for key, desc := range description {
				name := lipgloss.NewStyle().Bold(true).Render(key)
				current := lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Render(fmt.Sprintf("%v", viper.Get(key)))
				description := lipgloss.NewStyle().Foreground(lipgloss.Color("#9d9c9c")).Render(desc)
				rows = append(rows, []string{name, current, description})
			}

			internal.PrintTable(headers, rows, true)
		}
		if len(args) == 1 {
			fmt.Printf("The setting \"%s\" is currently set to \"%v\".\n", args[0], viper.Get(args[0]))
			return
		}

		if len(args) == 2 {
			viper.Set(args[0], args[1])
			fmt.Printf("The setting \"%s\" has been set to \"%v\".\n", args[0], viper.Get(args[0]))
			return
		}
	},
}

var service = &cobra.Command{
	Use:   "service",
	Short: "Use drop service to install, start, kill, or uninstall Drop as a background service.",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			fmt.Printf("%s You forgot to specify a command! Use \"drop service [install/start/kill/uninstall] or [i/s/k/u]\" to manage the Drop daemon.\n", internal.Icons.Negative)
			return
		}

		if slices.Contains([]string{"install", "i", "start", "s", "kill", "k", "uninstall", "u", "internal-run", "win-start"}, args[0]) {
			action := args[0]
			err := daemon.Execute(action)
			if err != nil {
				fmt.Printf("%s Ran into a problem: %v\n", internal.Icons.Negative, err)
				return
			}
			fmt.Printf("%s Successfully %s the Drop daemon. You can close this terminal now.\n", internal.Icons.Positive, func() string {
				switch action {
				case "install", "i":
					return "installed and started"
				case "start", "s":
					return "started"
				case "kill", "k":
					return "stopped"
				case "uninstall", "u":
					return "stopped and uninstalled"
				default:
					return ""
				}
			}())
		} else {
			fmt.Printf("%s Unknown command \"%s\". Use \"drop service [install/start/kill/uninstall] or [i/s/k/u]\" to manage the Drop daemon.\n", internal.Icons.Negative, args[0])
		}
	},
}

func init() {
	rootCmd.AddCommand(list, share, config, service)
	share.Flags().BoolVarP(&textMode, "text", "t", false, "Share a text snippet instead of a file.")
	share.Flags().BoolVarP(&dirMode, "dir", "d", false, "Share an entire directory instead of a file.")

}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		fmt.Printf("%s Error: %v\n", internal.Icons.Negative, err)
	}
}
