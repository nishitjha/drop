package internal

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

func StreamFile(deviceAddress string, deviceName string, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("Error opening file: %v", err)
	}
	defer file.Close()

	pr, pw := io.Pipe()
	bodyWriter := multipart.NewWriter(pw)

	contentType := bodyWriter.FormDataContentType()
	
	go func() {
		defer pw.Close()
		defer bodyWriter.Close()
		
		fileWriter, err := bodyWriter.CreateFormFile("file", filepath.Base(filePath))
		if err != nil {
			return
		}

		// I think a 1MB buffer is the right size "objectively"
		// TODO: add a user-facing setting for buffer size but yeah default to 1MB 
		buf := make([]byte, 1024*1024)  
		_, _ = io.CopyBuffer(fileWriter, file, buf)
	}()

	httpClient := &http.Client{}
	req, _ := http.NewRequest("POST", fmt.Sprintf("http://%s:3000/upload", deviceAddress), pr)
	req.Header.Set("Content-Type", contentType)

	response, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("Error sending file: %v", err)
	}

	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("Failed to send file. Status code: %d", response.StatusCode)
	}

	return nil
}