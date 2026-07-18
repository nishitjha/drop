package webserver

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
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
	Message string
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
}

type confirmModel struct {
	req      AuthRequest
	choice   bool
	answered bool
	quit     bool
}

var incomingRequests = make(chan AuthRequest)

var pendingMu sync.RWMutex
var pendingRequests = make(map[string]chan bool)

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
			location := filepath.Join(receiveDir, fmt.Sprintf("text_snippet_%s.txt", timestamp))

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
			context.JSON(200, JSONresponse{Message: "Text snippet sent successfully."})
			return
		}

		fileName := context.GetHeader("X-Filename")
		location := filepath.Join(receiveDir, fileName)

		err := os.MkdirAll(receiveDir, os.ModePerm)
		if err != nil {
			fmt.Println(err)
			context.JSON(500, JSONresponse{Message: err.Error()})
			return
		}

		internal.RunSpinner("Getting your file..", func() tea.Msg {
			out, err := os.Create(location)
			if err != nil {
				fmt.Println(err)
				context.JSON(500, JSONresponse{Message: err.Error()})
				return internal.TaskResultMsg{}
			}
			defer out.Close()

			_, err = io.CopyBuffer(out, context.Request.Body, make([]byte, 1024*1024))
			if err != nil {
				fmt.Println(err)
				context.JSON(500, JSONresponse{Message: err.Error()})
				return internal.TaskResultMsg{}
			}

			return internal.TaskResultMsg{}
		})

		context.JSON(200, JSONresponse{Message: fmt.Sprintf("File %s uploaded successfully", fileName)})
		fmt.Printf("%s Received file %s. You can find it at %s.\n", internal.Icons.Positive, fileName, location)
	})

	router.GET("/request", func(context *gin.Context) {
		senderName := context.Query("senderName")
		senderUUID := context.Query("UUID")
		fileName := context.Query("fileName")
		fileSize := context.Query("fileSize")
		textMode := context.Query("t") == "true"
		directoryMode := context.Query("d") == "true"

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
		}

		select {
		case accepted := <-answerChan:
			if accepted {
				context.JSON(200, JSONresponse{Message: "Accepted"})
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
		fileName := context.Query("fileName")

		html := fmt.Sprintf(`
<html>
<body>
<h2>%s wants to share %s with you.</h2>
<form action="/reqweb" method="POST">
<input type="hidden" name="id" value="%s">
<button type="submit" name="answer" value="true">Yes</button>
<button type="submit" name="answer" value="false">No</button>
</form>
</body>
</html>
`, senderName, fileName, id)

		context.Header("Content-Type", "text/html")
		context.String(200, html)
	})

	router.POST("/reqweb", func(context *gin.Context) {
		id := context.PostForm("id")
		answer := context.PostForm("answer") == "true"

		answerChan, ok := popPending(id)
		if !ok {
			context.JSON(404, JSONresponse{Message: "Request not found or already answered"})
			return
		}

		answerChan <- answer
		context.String(200, "You can close this tab now.")
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

			archivePath := filepath.Join(receiveDir, fileName)

			internal.RunSpinner("Getting your folder..", func() tea.Msg {
				out, err := os.Create(archivePath)
				if err != nil {
					fmt.Printf("%s Error creating archive file: %v\n", internal.Icons.Negative, err)
					return internal.TaskResultMsg{}
				}

				_, err = io.CopyBuffer(out, context.Request.Body, make([]byte, 1024*1024))
				if err != nil {
					fmt.Printf("%s Error saving archive: %v\n", internal.Icons.Negative, err)
					return internal.TaskResultMsg{}
				}
				out.Close()

				return internal.TaskResultMsg{}
			})

			if autoExtract {
				err := archive.ExtractArchive(archivePath, receiveDir, fileName)
				if err != nil {
					fmt.Printf("%s Error extracting archive: %v\n", internal.Icons.Negative, err)
					context.JSON(500, JSONresponse{Message: "Something went wrong."})
					return
				} else {
					fmt.Printf("%s Successfully extracted archived folder to %s.\n", internal.Icons.Positive, filepath.Join(receiveDir, strings.TrimSuffix(fileName, "_drop.zip")))
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
			context.JSON(200, JSONresponse{Message: fmt.Sprintf("Archive %s uploaded successfully", fileName)})

			return
		}

	})

	err := router.Run(fmt.Sprintf("0.0.0.0:%d", viper.GetInt("webserver.port")))
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
		case "left", "right", "up", "down", "tab":
			m.choice = !m.choice
		case "enter", "space":
			m.answered = true
			return m, tea.Quit
		}
		switch msg.String() {
		case "y":
			m.choice = true
			m.answered = true
			return m, tea.Quit
		case "n":
			m.choice = false
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
	s += " Use the arrow keys to select an option and press enter to confirm. \n You may also press the keys 'y' or 'n' to accept or decline respectively. \n\n"
	s += " You have three minutes to respond. \n\n"
	if m.req.FileName != "" {
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

	if m.choice {
		s += "  [ Yes ]    No  \n\n"
	} else {
		s += "    Yes    [ No ]\n\n"
	}

	return tea.NewView(s)
}

func HandleRequests(mode string) {
	for req := range incomingRequests {

		if mode == "daemon" {
			browser.OpenURL(fmt.Sprintf("http://localhost:%d/reqweb?id=%s&senderName=%s&fileName=%s&fileSize=%d&t=%t&d=%t",
				viper.GetInt("webserver.port"),
				req.RequestID,
				req.SenderName,
				req.FileName,
				req.FileSize,
				req.TextMode,
				req.DirectoryMode))

		} else {
			m := confirmModel{
				req:    req,
				choice: true,
			}

			p := tea.NewProgram(m)
			finalModel, err := p.Run()

			if err != nil {
				req.Response <- false
				continue
			}

			fm := finalModel.(confirmModel)

			if fm.answered && fm.choice {
				fmt.Printf("%s Accepted sharing request from \"%s\".\n", internal.Icons.Positive, req.SenderName)
				req.Response <- true
			} else {
				fmt.Printf("%s Declined sharing request from \"%s\".\n", internal.Icons.Negative, req.SenderName)
				req.Response <- false
			}
		}
	}

}