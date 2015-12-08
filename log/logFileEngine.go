package log

import (
	"os"
	"log"
	"fmt"
)

type FileClient struct {
	logFile 		string
	copyToStdOut 	bool
}

func StartFileClient(path string, copyToStdout bool) *FileClient {
	return &FileClient{path, copyToStdout}
}

func (fc *FileClient) WriteMessage(message string) error {
	f, err := os.OpenFile(fc.logFile, os.O_RDWR | os.O_CREATE | os.O_APPEND, 0666)
	if err != nil {
	    panic("log : error opening file.")
	}
	defer f.Close()

	log.SetOutput(f)
	log.Println(message)

	if fc.copyToStdOut {
		fmt.Println(message)
	}

	return nil
}