package internal

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	tea "charm.land/bubbletea/v2"
)

type ProgressWriter struct {
	TotalBytes int64
	Err        error
	Program   *tea.Program
	FileSize   int64
}

func (progWriter *ProgressWriter) Write(p []byte) (n int, err error) {
	progWriter.TotalBytes += int64(len(p))
	progWriter.Program.Send(progressMsg{Decimal: float64(progWriter.TotalBytes) / float64(progWriter.FileSize)})
	return len(p), nil
}

/*func StreamFile(deviceAddress string, deviceName string, filePath string) error {
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
}*/

func StreamFile(deviceAddress string, deviceName string, filePath string, program *tea.Program) error {
    file, err := os.Open(filePath)
    if err != nil {
        program.Send(doneMsg{Err: err})
        return fmt.Errorf("%s Error opening file: %v", Icons.Negative, err)
    }
    defer file.Close()
    
    fileInfo, err := os.Stat(filePath)
    if err != nil {
        program.Send(doneMsg{Err: err})
        return fmt.Errorf("%s Error opening file \"%s\": %v\n", Icons.Negative, filePath, err)
    }
    
    reader := io.TeeReader(file, &ProgressWriter{TotalBytes: 0, FileSize: fileInfo.Size(), Program: program})
    req, err := http.NewRequest("POST", fmt.Sprintf("http://%s:3000/upload", deviceAddress), reader)
    if err != nil {
        program.Send(doneMsg{Err: err})
        return fmt.Errorf("%s Error creating request: %v", Icons.Negative, err)
    }

    req.Header.Set("X-Filename", filepath.Base(filePath))
    req.Header.Set("X-Filesize", fmt.Sprintf("%d", fileInfo.Size()))
    req.ContentLength = fileInfo.Size()
    httpClient := &http.Client{}
    response, err := httpClient.Do(req)

    if err != nil {
        program.Send(doneMsg{Err: err})
        return fmt.Errorf("%s Error sending file: %v", Icons.Negative, err)
    }
    defer response.Body.Close()

    if response.StatusCode != http.StatusOK {
        fmt.Printf("%s Failed to send file. Status code: %d", Icons.Negative, response.StatusCode)
        program.Send(doneMsg{Err: fmt.Errorf("Failed to send file. Status code: %d", response.StatusCode)})
    } else if response.StatusCode == http.StatusOK {
        fmt.Printf("\r\033[2K%s The file \"%s\" has been sent successfully to %s.\n", Icons.Positive, filepath.Base(filePath), deviceName)
        
    program.Send(doneMsg{Err: err})
    }
    return nil
}