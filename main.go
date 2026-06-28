package main

import (
	"github.com/gin-gonic/gin"
	"github.com/professional-procrastinator/drop/devices"
)

type response struct {
	Message string 
} 

func main() {
	router := gin.Default()
	

	router.GET("/upload", func(context *gin.Context) {
		context.JSON(
			200,
			response{
				Message: "hi!",
			})
	})

	go devices.LinkBLE() //goroutine
	err := router.Run("0.0.0.0:3001") 
	if err != nil {
		panic(err)
	}

}