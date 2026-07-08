package webserver

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gin-gonic/gin"
	"github.com/nishitjha/drop/internal"
	"github.com/spf13/viper"
	"golang.design/x/clipboard"
	//"github.com/ncruces/zenity"
	//"github.com/sqweek/dialog"
)

type JSONresponse struct {
	Message string
}

type AuthRequest struct {
	SenderName string
	SenderUUID string
	Response   chan bool
	FileName   string
	FileSize   int64
	TextMode   bool
}

type confirmModel struct {
	req      AuthRequest
	choice   bool
	answered bool
	quit     bool
}

var incomingRequests = make(chan AuthRequest)

func Listen() {
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()

	go HandleRequests()

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

		out, err := os.Create(location)
		if err != nil {
			fmt.Println(err)
			context.JSON(500, JSONresponse{Message: err.Error()})
			return
		}
		defer out.Close()

		_, err = io.CopyBuffer(out, context.Request.Body, make([]byte, 1024*1024))
		
		if err != nil {
			fmt.Println(err)
			context.JSON(500, JSONresponse{Message: err.Error()})
			return
		}
		context.JSON(200, JSONresponse{Message: fmt.Sprintf("File %s uploaded successfully", fileName)})
		fmt.Printf("%s Received file %s. You can find it at %s.\n", internal.Icons.Positive, fileName, location)
	})

	router.GET("/request", func(context *gin.Context) {
		senderName := context.Query("senderName")
		senderUUID := context.Query("UUID")
		fileName := context.Query("fileName")
		fileSize := context.Query("fileSize")
		textMode := context.Query("t") == "true"

		answerChan := make(chan bool)

		incomingRequests <- AuthRequest{
			SenderName: senderName,
			SenderUUID: senderUUID,
			Response:   answerChan,
			FileName:   fileName,
			FileSize: func() int64 {
				var size int64
				fmt.Sscanf(fileSize, "%d", &size)
				return size
			}(),
			TextMode: textMode,
		}

		select {
		case accepted := <-answerChan:
			if accepted {
				context.JSON(200, JSONresponse{Message: "Accepted"})
			} else {
				context.JSON(403, JSONresponse{Message: "Declined"})
			}
		case <-time.After(3 * time.Minute):
			context.JSON(403, JSONresponse{Message: "Timeout"})
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
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.quit = true
			return m, tea.Quit
		case "left", "right", "up", "down", "tab":
			m.choice = !m.choice
		case "enter", " ":
			m.answered = true
			return m, tea.Quit
		}
		// Allow 'y' and 'n' keys for quick response
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

func (m confirmModel) View() string {
	if m.answered || m.quit {
		return ""
	}

	s := fmt.Sprintf("\n Do you wish to accept a %s sharing request from \"%s\"?\n\n", func() string {
		if m.req.TextMode {
			return "text"
		}
		return "file"
	}(), m.req.SenderName)
	s += " Use the arrow keys to select an option and press enter to confirm. \n You may also press the keys 'y' or 'n' to accept or decline respectively. \n\n"
	s += " You have three minutes to respond. \n\n"
 	if m.req.FileName != "" {
		s += fmt.Sprintf(" - File name: %s\n", m.req.FileName)
		s += fmt.Sprintf(" - File size: %d bytes\n\n", m.req.FileSize)
	}

	if m.choice {
		s += "  [ Yes ]    No  \n\n"
	} else {
		s += "    Yes    [ No ]\n\n"
	}

	return s
}

func HandleRequests() {
	for req := range incomingRequests {
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
			fmt.Printf("Accepted sharing request from \"%s\".\n", req.SenderName)
			req.Response <- true
		} else {
			fmt.Printf("Declined sharing request from \"%s\".\n", req.SenderName)
			req.Response <- false
		}
	}
}