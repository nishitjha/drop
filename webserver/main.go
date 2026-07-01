package webserver

import (
	"bufio"
	"fmt"
	"os"
	"time"

	"github.com/briandowns/spinner"
	"github.com/gin-gonic/gin"
)

type JSONresponse struct {
	Message string
}

func Listen() {
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()

	router.POST("/upload", func(context *gin.Context) {
		file, err := context.FormFile("file")

		if err != nil {
			fmt.Println(err)
			return
		}

		context.SaveUploadedFile(file, "../uploads/"+file.Filename)
		context.JSON(200, JSONresponse{Message: fmt.Sprintf("File %s uploaded successfully", file.Filename)})
		fmt.Printf("Received file %s.", file.Filename)
	})

	// make router devices cache endpoint

	router.GET("/request", func(context *gin.Context) {
		senderName := context.Query("senderName")
		senderUUID := context.Query("UUID")
		

		if SharingRequest(senderName, senderUUID) {
			response := JSONresponse{Message: "Accepted"}
			context.JSON(200, response)
		} else if !SharingRequest(senderName, senderUUID) {
			response := JSONresponse{Message: "Declined"}
			context.JSON(403, response)
		}

	}) 

	err := router.Run("0.0.0.0:3000")
	if err != nil {
		panic(err)
	}
}

func SharingRequest(senderName string, senderUUID string) (choice bool) {
	// time out after 3 minutes if the user doesn't respond
    reader := bufio.NewReader(os.Stdin)

	sp := spinner.New(spinner.CharSets[40], 100*time.Millisecond)
	sp.Suffix = fmt.Sprintf(" Incoming sharing request from \"%[1]s\".", senderName)
	sp.Start()

	fmt.Println("You have 3 minutes to respond. Accept the request by typing 'yes'/'y' or decline by typing 'no'/'n'.")
    
	answer, _ := reader.ReadString('\n')
	if answer == "yes\n" || answer == "y\n" {
		fmt.Printf("Accepted sharing request from \"%[1]s\".", senderName)
		return true
	}	
	fmt.Printf("Declined sharing request from \"%[1]s\".", senderName)
	return false

}