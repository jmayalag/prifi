package log

import (
	"os"
	"fmt"
	"strings"
)

type FileClient struct {
	logFile 		string
	copyToStdOut 	bool
}

func StartFileClient(path string, copyToStdout bool) *FileClient {
	return &FileClient{path, copyToStdout}
}

func (fc *FileClient) WriteMessage(message string) error {
	f, err := os.OpenFile(fc.logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
	    panic(err)
	}

	defer f.Close()

	if _, err = f.WriteString(message); err != nil {
	    panic(err)
	}

	if fc.copyToStdOut && !strings.HasPrefix(message, "{") {
		fmt.Println(message)
	}

	return nil
}