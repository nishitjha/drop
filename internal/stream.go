package internal

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

func StreamFile(deviceAddress string, deviceName string, filePath string) {
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		return
	}
	defer file.Close()

	
	bodyBuf := &bytes.Buffer{}
	bodyWriter := multipart.NewWriter(bodyBuf)
	fileWriter, err := bodyWriter.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		fmt.Printf("Error creating form file: %v\n", err)
		return
	}

	_, err = io.Copy(fileWriter, file)
	if err != nil {
		fmt.Printf("Error copying file: %v\n", err)
		return
	}
	bodyWriter.Close()

	httpClient := &http.Client{ // skip the timeout for now 
	}	
	req, _ := http.NewRequest("POST", fmt.Sprintf("http://%s:3000/upload", deviceAddress), bodyBuf)
	req.Header.Set("Content-Type", bodyWriter.FormDataContentType())

	response , err := httpClient.Do(req)
	if err != nil {
		fmt.Printf("Error sending file: %v\n", err)
		return
	}

	defer response.Body.Close()
	if response.StatusCode == http.StatusOK {
		fmt.Printf("File \"%[1]s\" sent successfully to %[2]s.\n", filepath.Base(filePath), deviceName)
	} else {
		fmt.Printf("Failed to send file. Status code: %d\n", response.StatusCode)
	}
}