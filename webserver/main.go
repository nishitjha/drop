package webserver

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

type JSONresponse struct {
	Message string
}

func Listen() {
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

	err := router.Run("0.0.0.0:3000")
	if err != nil {
		panic(err)
	}
}