package webserver

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gin-gonic/gin"
	"github.com/ncruces/zenity"
)

type JSONresponse struct {
	Message string
}

type AuthRequest struct {
	SenderName string
	SenderUUID string
	Response   chan bool
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
		file, err := context.FormFile("file")
		if err != nil {
			fmt.Println(err)
			return
		}

		context.SaveUploadedFile(file, "../uploads/"+file.Filename)
		context.JSON(200, JSONresponse{Message: fmt.Sprintf("File %s uploaded successfully", file.Filename)})
		fmt.Printf("Received file %s.\n", file.Filename)
	})

	router.GET("/request", func(context *gin.Context) {
		senderName := context.Query("senderName")
		senderUUID := context.Query("UUID")

		answerChan := make(chan bool)

		incomingRequests <- AuthRequest{
			SenderName: senderName,
			SenderUUID: senderUUID,
			Response:   answerChan,
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

	err := router.Run("0.0.0.0:3000")
	if err != nil {
		panic(err)
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
	}
	return m, nil
}

func (m confirmModel) View() string {
	if m.answered || m.quit {
		return ""
	}

	s := fmt.Sprintf("\n Do you wish to accept a sharing request from \"%s\"?\n", m.req.SenderName)
	s += " Use the arrow keys to select an option and press enter to confirm. You have three minutes to respond.\n\n"
	if m.choice {
		s += "  [ Yes ]    No  \n\n"
	} else {
		s += "    Yes    [ No ]\n\n"
	}

	return s
}

func HandleRequests() {
	for req := range incomingRequests {
		/*m := confirmModel{
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
		}*/
		err := zenity.Question(fmt.Sprintf("Do you wish to accept a sharing request from \"%s\"? You have three minutes to respond.", req.SenderName), zenity.Title("Incoming Sharing Request"), zenity.OKLabel("Accept"), zenity.CancelLabel("Decline"))
		if err != nil {
			fmt.Printf("Declined sharing request from \"%s\".\n", req.SenderName)
			req.Response <- false
		} else {
			fmt.Printf("Accepted sharing request from \"%s\".\n", req.SenderName)
			req.Response <- true
		}
	}
}