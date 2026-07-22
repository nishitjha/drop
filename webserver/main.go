package webserver

import (
	"embed"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/nishitjha/drop/internal"
	"github.com/nishitjha/drop/internal/archive"
	"github.com/pkg/browser"
	"github.com/spf13/viper"
	"golang.design/x/clipboard"
)

type JSONresponse struct {
	Message   string
	RequestID string `json:",omitempty"`
}

type AuthRequest struct {
	RequestID     string
	SenderName    string
	SenderUUID    string
	Response      chan bool
	FileName      string
	FileSize      int64
	TextMode      bool
	DirectoryMode bool
	Trusted       bool
	Context       *gin.Context
}

type confirmModel struct {
	req       AuthRequest
	selection int
	answered  bool
	quit      bool
}

var incomingRequests = make(chan AuthRequest)

var pendingMu sync.RWMutex
var pendingRequests = make(map[string]chan bool)

var transferMu sync.RWMutex
var transferPaths = make(map[string]string)

//go:embed web
var webFS embed.FS
var reqwebTemplate = template.Must(
	template.New("request.html").
		Funcs(template.FuncMap{"formatBytes": internal.FormatBytes}).
		ParseFS(webFS, "web/request.html"),
)

var resultTemplate = template.Must(
	template.New("result.html").ParseFS(webFS, "web/result.html"),
)

func addPending(id string, ch chan bool) {
	pendingMu.Lock()
	defer pendingMu.Unlock()
	pendingRequests[id] = ch
}

func popPending(id string) (chan bool, bool) {
	pendingMu.Lock()
	defer pendingMu.Unlock()
	ch, ok := pendingRequests[id]
	if ok {
		delete(pendingRequests, id)
	}
	return ch, ok
}

func Listen(mode string) {
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	go HandleRequests(mode)

	webSub, err := fs.Sub(webFS, "web")
	if err != nil {
		fmt.Println(err)
	}
	router.StaticFS("/static", http.FS(webSub))
	// router.SetFuncMap(template.FuncMap{
	// 	"formatBytes": internal.FormatBytes,
	// })
	// router.LoadHTMLGlob("../web/*")

	router.POST("/upload", func(context *gin.Context) {
		receiveDir := viper.GetString("sharing.receiveDir")
		askEverytime := viper.GetBool("sharing.askReceiveDirEverytime")

		if askEverytime {
			// i'll ask them i promise
		}

		isTextSnippet := context.GetHeader("X-TextSnippet") == "true"

		if isTextSnippet {
			copyToClipboard := viper.GetBool("sharing.autoCopyToClipboard")

			if copyToClipboard {
				err := clipboard.Init()
				if err != nil {
					fmt.Printf("%s Error initializing clipboard: %s \n", internal.Icons.Negative, err)
					context.JSON(500, JSONresponse{Message: "Something went wrong."})
					return
				}

				bodyBytes, err := io.ReadAll(context.Request.Body)
				if err != nil {
					fmt.Printf("%s Error reading request body: %v\n", internal.Icons.Negative, err)
					context.JSON(500, JSONresponse{Message: "Something went wrong."})
					return
				}

				clipboard.Write(clipboard.FmtText, bodyBytes)

				fmt.Printf("%s The text snippet has been copied to your clipboard.\n", internal.Icons.Positive)

				context.JSON(200, JSONresponse{Message: "Text snippet sent successfully."})

				return
			}
			timestamp := time.Now().Format("02-01-06")
			location := uniquePath(filepath.Join(receiveDir, fmt.Sprintf("text_snippet_%s.txt", timestamp)))

			err := os.MkdirAll(receiveDir, os.ModePerm)
			if err != nil {
				fmt.Printf("%s Error creating directory: %v\n", internal.Icons.Negative, err)
				context.JSON(500, JSONresponse{Message: "Something went wrong."})
				return
			}

			file, err := os.Create(location)
			if err != nil {
				fmt.Printf("%s Error creating text snippet file: %v\n", internal.Icons.Negative, err)
				context.JSON(500, JSONresponse{Message: "Something went wrong."})
				return
			}
			defer file.Close()

			_, err = io.Copy(file, context.Request.Body)
			if err != nil {
				fmt.Printf("%s Error saving text snippet: %v\n", internal.Icons.Negative, err)
				context.JSON(500, JSONresponse{Message: "Something went wrong."})
				return
			}

			fmt.Printf("%s The text snippet has been saved at %s.\n", internal.Icons.Positive, location)
			if reqID := context.GetHeader("X-RequestID"); reqID != "" {
				transferMu.Lock()
				transferPaths[reqID] = location
				transferMu.Unlock()
			}
			context.JSON(200, JSONresponse{Message: "Text snippet sent successfully."})
			return
		}

		fileName := context.GetHeader("X-Filename")
		location := uniquePath(filepath.Join(receiveDir, fileName))

		err := os.MkdirAll(receiveDir, os.ModePerm)
		if err != nil {
			fmt.Println(err)
			context.JSON(500, JSONresponse{Message: err.Error()})
			return
		}

		writeFile := func() error {
			out, err := os.Create(location)
			if err != nil {
				return err
			}
			defer out.Close()

			_, err = io.CopyBuffer(out, context.Request.Body, make([]byte, 1024*1024))
			return err
		}

		if mode == "daemon" {
			err = writeFile()
		} else {
			result := internal.RunSpinner("Getting your file..", func() tea.Msg {
				return internal.TaskResultMsg{Error: writeFile()}
			})
			err = result.Error
		}

		if err != nil {
			fmt.Println(err)
			context.JSON(500, JSONresponse{Message: err.Error()})
			return
		}

		context.JSON(200, JSONresponse{Message: fmt.Sprintf("File %s uploaded successfully", fileName)})
		fmt.Printf("%s Received file %s. You can find it at %s.\n", internal.Icons.Positive, fileName, location)
		if reqID := context.GetHeader("X-RequestID"); reqID != "" {
			transferMu.Lock()
			transferPaths[reqID] = location
			transferMu.Unlock()
		}
	})

	router.GET("/request", func(context *gin.Context) {
		senderName := context.Query("senderName")
		senderUUID := context.Query("UUID")
		fileName := context.Query("fileName")
		fileSize := context.Query("fileSize")
		textMode := context.Query("t") == "true"
		directoryMode := context.Query("d") == "true"

		trustedUUIDs := viper.GetStringSlice("sharing.trustedDevices")
		var trustedSender bool = false
		if slices.Contains(trustedUUIDs, senderUUID) || viper.GetBool("sharing.trustAllDevices") {
			trustedSender = true
		}

		requestID := uuid.New().String()
		answerChan := make(chan bool)
		addPending(requestID, answerChan)

		incomingRequests <- AuthRequest{
			RequestID:  requestID,
			SenderName: senderName,
			SenderUUID: senderUUID,
			Response:   answerChan,
			FileName:   fileName,
			FileSize: func() int64 {
				var size int64
				fmt.Sscanf(fileSize, "%d", &size)
				return size
			}(),
			TextMode:      textMode,
			DirectoryMode: directoryMode,
			Trusted:       trustedSender,
			Context:       context,
		}

		select {
		case accepted := <-answerChan:
			if accepted {
				context.JSON(200, JSONresponse{Message: "Accepted", RequestID: requestID})
			} else {
				context.JSON(403, JSONresponse{Message: "Declined"})
			}
		case <-time.After(3 * time.Minute):
			popPending(requestID)
			context.JSON(403, JSONresponse{Message: "Timeout"})
		}
	})

	router.GET("/reqweb", func(context *gin.Context) {
		id := context.Query("id")
		senderName := context.Query("senderName")
		senderUUID := context.Query("senderUUID")
		fileName := context.Query("fileName")
		textMode := context.Query("t") == "true"
		directoryMode := context.Query("d") == "true"

		var fileSize int64
		fmt.Sscanf(context.Query("fileSize"), "%d", &fileSize)

		trustedUUIDs := viper.GetStringSlice("sharing.trustedDevices")
		trusted := slices.Contains(trustedUUIDs, senderUUID) || viper.GetBool("sharing.trustAllDevices")

		context.Header("Content-Type", "text/html")
		reqwebTemplate.Execute(context.Writer, gin.H{
			"title":         "Drop Sharing Request",
			"sender":        senderName,
			"senderUUID":    senderUUID,
			"fileName":      fileName,
			"fileSize":      fileSize,
			"textMode":      textMode,
			"directoryMode": directoryMode,
			"trusted":       trusted,
			"id":            id,
		})
	})
	router.POST("/reqweb", func(context *gin.Context) {
		id := context.PostForm("id")
		senderUUID := context.PostForm("senderUUID")
		textMode := context.PostForm("textMode") == "true"
		directoryMode := context.PostForm("directoryMode") == "true"
		answer := context.PostForm("answer") == "true"
		trust := context.PostForm("trust") == "true"

		answerChan, ok := popPending(id)
		if !ok {
			context.Header("Content-Type", "text/html")
			resultTemplate.Execute(context.Writer, gin.H{
				"heading":  "Something went wrong",
				"subtitle": "This request was not found, or has already been answered. Try sharing it again?",
				"pollable": false,
			})
			return
		}

		if answer && trust && senderUUID != "" {
			trustedUUIDs := viper.GetStringSlice("sharing.trustedDevices")
			if !slices.Contains(trustedUUIDs, senderUUID) {
				trustedUUIDs = append(trustedUUIDs, senderUUID)
				viper.Set("sharing.trustedDevices", trustedUUIDs)
				viper.WriteConfig()
			}
		}

		answerChan <- answer

		context.Header("Content-Type", "text/html")

		if !answer {
			resultTemplate.Execute(context.Writer, gin.H{
				"heading":  "Declined",
				"subtitle": "You can close this tab now.",
				"pollable": false,
			})
			return
		}

		subtitle := fmt.Sprintf("You'll receive your %s in... just kidding IDK how much time it'll take. \n You can close this tab or wait till an \"Open\" button appears here.", func() string {
			if !textMode && !directoryMode {
				return "file"
			} else if textMode {
				return "text snippet"
			} else {
				return "folder"
			}
		}())
		if textMode {
			if viper.GetBool("sharing.autoCopyToClipboard") {
				subtitle = "The text will be copied to your clipboard."
			} else {
				subtitle = "The text will be saved to a text file."
			}
		}

		resultTemplate.Execute(context.Writer, gin.H{
			"heading":  "Accepted",
			"subtitle": subtitle,
			"pollable": !textMode,
			"id":       id,
		})
	})

	router.GET("/reqweb/status", func(context *gin.Context) {
		id := context.Query("id")
		transferMu.RLock()
		path, done := transferPaths[id]
		transferMu.RUnlock()
		context.JSON(200, gin.H{"done": done, "path": path})
	})

	router.GET("/reveal", func(context *gin.Context) {
		path := context.Query("path")
		if path == "" {
			context.String(400, "Missing path")
			return
		}
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "darwin":
			cmd = exec.Command("open", "-R", path)
		case "windows":
			cmd = exec.Command("explorer", "/select,", path)
		default:
			cmd = exec.Command("xdg-open", filepath.Dir(path))
		}
		cmd.Start()
		context.String(200, "Opened.")
	})
	router.POST("/archive", func(context *gin.Context) {
		receiveDir := viper.GetString("sharing.receiveDir")
		askEverytime := viper.GetBool("sharing.askReceiveDirEverytime")
		autoExtract := viper.GetBool("sharing.folders.autoExtractOnReceive")
		format := context.Query("format")

		fileName := context.GetHeader("X-Filename")

		if askEverytime {
			// i'll ask them i promise
		}

		if format != "zip" && format != "tar.gz" {
			context.JSON(400, JSONresponse{Message: "Invalid format. Use 'zip' or 'tar.gz'."})
			return
		}

		if format == "zip" {
			err := os.MkdirAll(receiveDir, os.ModePerm)
			if err != nil {
				fmt.Printf("%s Error creating directory: %v\n", internal.Icons.Negative, err)
				context.JSON(500, JSONresponse{Message: "Something went wrong."})
				return
			}

			archivePath := uniquePath(filepath.Join(receiveDir, fileName))

			writeArchive := func() error {
				out, err := os.Create(archivePath)
				if err != nil {
					return err
				}
				defer out.Close()

				_, err = io.CopyBuffer(out, context.Request.Body, make([]byte, 1024*1024))
				return err
			}

			if mode == "daemon" {
				err = writeArchive()
			} else {
				result := internal.RunSpinner("Getting your folder..", func() tea.Msg {
					return internal.TaskResultMsg{Error: writeArchive()}
				})
				err = result.Error
			}

			if err != nil {
				fmt.Printf("%s Error saving archive: %v\n", internal.Icons.Negative, err)
				context.JSON(500, JSONresponse{Message: "Something went wrong."})
				return
			}

			if autoExtract {
				extractDir := uniquePath(filepath.Join(receiveDir, strings.TrimSuffix(fileName, "_drop.zip")))
				err := archive.ExtractArchive(archivePath, extractDir, fileName)
				if err != nil {
					fmt.Printf("%s Error extracting archive: %v\n", internal.Icons.Negative, err)
					context.JSON(500, JSONresponse{Message: "Something went wrong."})
					return
				} else {
					fmt.Printf("%s Successfully extracted archived folder to %s.\n", internal.Icons.Positive, extractDir)
					if reqID := context.GetHeader("X-RequestID"); reqID != "" {
						transferMu.Lock()
						transferPaths[reqID] = extractDir
						transferMu.Unlock()
					}
				}

				context.JSON(200, JSONresponse{Message: fmt.Sprintf("Archive %s uploaded and extracted successfully", fileName)})

				err = os.Remove(archivePath)
				if err != nil {
					fmt.Printf("%s Could not delete the archive after extraction (not fatal). Try deleting it yourself.\n", internal.Icons.Warning)
					return
				}
				return
			}

			fmt.Printf("%s Received archive %s. You can find it at %s.\n", internal.Icons.Positive, fileName, archivePath)
			if reqID := context.GetHeader("X-RequestID"); reqID != "" {
				transferMu.Lock()
				transferPaths[reqID] = archivePath
				transferMu.Unlock()
			}
			context.JSON(200, JSONresponse{Message: fmt.Sprintf("Archive %s uploaded successfully", fileName)})

			return
		}

	})

	err = router.Run(fmt.Sprintf("0.0.0.0:%d", viper.GetInt("webserver.port")))
	if err != nil {
		fmt.Printf("%s Error starting web server: %v\n", internal.Icons.Negative, err)
	}
}

func (m confirmModel) Init() tea.Cmd {
	return nil
}

func (m confirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.quit = true
			return m, tea.Quit
		case "left", "up":
			m.selection = (m.selection + 2) % 3
		case "right", "down", "tab":
			m.selection = (m.selection + 1) % 3
		case "enter", "space":
			m.answered = true
			return m, tea.Quit
		}
		switch msg.String() {
		case "y":
			m.selection = 1
			m.answered = true
			return m, tea.Quit
		case "n":
			m.selection = 0
			m.answered = true
			return m, tea.Quit
		case "t":
			m.selection = 2
			m.answered = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m confirmModel) View() tea.View {
	if m.answered || m.quit {
		return tea.NewView("")
	}
	s := fmt.Sprintf("\n Do you wish to accept a %s sharing request from \"%s\"?\n\n", func() string {
		if m.req.TextMode {
			return internal.TextStyle.Render("text")
		} else if m.req.DirectoryMode {
			return internal.FolderStyle.Render("folder")
		}
		return internal.FileStyle.Render("file")
	}(), m.req.SenderName)
	s += " Use the arrow keys to select an option and press enter to confirm. \n You may also press the keys 'y', 'n', or 't' to accept, decline, or trust this device respectively. \n\n"
	s += " You have three minutes to respond. \n\n"
	if m.req.TextMode {
		s += fmt.Sprintf(" - Text size: %d chars\n\n", m.req.FileSize)
	} else if m.req.FileName != "" {
		s += fmt.Sprintf(" - %s name: %s\n", func() string {
			if m.req.DirectoryMode {
				return "Folder"
			}
			return "File"
		}(), m.req.FileName)
		s += fmt.Sprintf(" - %s size: %s\n\n", func() string {
			if m.req.DirectoryMode {
				return "Folder"
			}
			return "File"
		}(), func() string {
			if m.req.DirectoryMode {
				return "unknown"
			}
			return internal.FormatBytes(m.req.FileSize)
		}())
	}

	s += fmt.Sprintf("  %s    %s    %s  \n\n", func() string {
		if m.selection == 1 {
			return "[ Yes ]"
		}
		return "  Yes  "
	}(), func() string {
		if m.selection == 0 {
			return "[ No ]"
		}
		return "  No  "
	}(), func() string {
		if m.selection == 2 {
			return "[ Trust this device ]"
		}
		return "  Trust this device  "
	}())

	return tea.NewView(s)
}

func HandleRequests(mode string) {
	for req := range incomingRequests {

		if mode == "daemon" {
			if req.Trusted {
				answerChan, ok := popPending(req.RequestID)
				if !ok {
					req.Context.JSON(404, JSONresponse{Message: "Request not found or already answered"})
					continue
				}
				answerChan <- true
				continue
			}

			browser.OpenURL(fmt.Sprintf("http://localhost:%d/reqweb?id=%s&senderName=%s&senderUUID=%s&fileName=%s&fileSize=%d&t=%t&d=%t",
				viper.GetInt("webserver.port"),
				req.RequestID,
				req.SenderName,
				req.SenderUUID,
				req.FileName,
				req.FileSize,
				req.TextMode,
				req.DirectoryMode))
		} else {
			if req.Trusted {
				fmt.Printf("%s Automatically accepted sharing request from trusted device \"%s\".\n", internal.Icons.Positive, req.SenderName)
				if req.TextMode {
					fmt.Printf(" - Text size: %d chars\n\n", req.FileSize)
				} else if req.FileName != "" {
					fmt.Printf(" - %s name: %s\n", func() string {
						if req.DirectoryMode {
							return "Folder"
						}
						return "File"
					}(), req.FileName)
					fmt.Printf(" - %s size: %s\n\n", func() string {
						if req.DirectoryMode {
							return "Folder"
						}
						return "File"
					}(), func() string {
						if req.DirectoryMode {
							return "unknown"
						}
						return internal.FormatBytes(req.FileSize)
					}())
				}
			}
			m := confirmModel{
				req:       req,
				selection: 1,
			}

			p := tea.NewProgram(m)
			finalModel, err := p.Run()

			if err != nil {
				req.Response <- false
				continue
			}

			fm := finalModel.(confirmModel)

			switch {
			case fm.answered && fm.selection == 1:
				fmt.Printf("%s Accepted sharing request from \"%s\".\n", internal.Icons.Positive, req.SenderName)
				req.Response <- true
			case fm.answered && fm.selection == 2:
				fmt.Printf("%s Accepted sharing request from \"%s\" and added them to your trusted devices.\n", internal.Icons.Positive, req.SenderName)
				req.Response <- true

				trustedUUIDs := viper.GetStringSlice("sharing.trustedDevices")
				if !slices.Contains(trustedUUIDs, req.SenderUUID) {
					trustedUUIDs = append(trustedUUIDs, req.SenderUUID)
					viper.Set("sharing.trustedDevices", trustedUUIDs)
					if err := viper.WriteConfig(); err != nil {
						fmt.Printf("%s Error saving trusted devices: %v\n", internal.Icons.Negative, err)
					}
				}
			default:
				fmt.Printf("%s Declined sharing request from \"%s\".\n", internal.Icons.Negative, req.SenderName)
				req.Response <- false
			}
		}
	}

}

func uniquePath(path string) string {
	if !viper.GetBool("sharing.autoRenameExistingFiles") {
		return path
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return path
	}

	dir := filepath.Dir(path)
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)

	for i := 1; ; i++ {
		candidate := filepath.Join(dir, fmt.Sprintf("%s (%d)%s", name, i, ext))
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}
}
