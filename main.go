package main

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/professional-procrastinator/drop/link"
)

type JSONresponse struct {
	Message string
}

func main() {
	router := gin.Default()

	router.POST("/upload", func(context *gin.Context) {
		file, err := context.FormFile("file")

		if err != nil {
			fmt.Println(err)
			return
		}

		context.SaveUploadedFile(file, "./uploads/"+file.Filename)
		context.JSON(200, JSONresponse{Message: fmt.Sprintf("File %s uploaded successfully", file.Filename)})
	})

	go link.LaunchService() //goroutine
	go link.ServiceBrowser()
	err := router.Run("0.0.0.0:3000")
	if err != nil {
		panic(err)
	}

}
