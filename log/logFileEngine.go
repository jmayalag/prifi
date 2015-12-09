package log

import (
	"os"
	"fmt"
	"strings"
)

type FileClient struct {
	logFile 		string
	copyToStdOut 	bool
	logLevel		int
}

func StartFileClient(logLevel int, path string, copyToStdout bool) *FileClient {
	return &FileClient{path, copyToStdout, logLevel}
}

func (fc *FileClient) WriteMessage(severity int, message string) error {

	if severity > fc.logLevel { //unintuitive : severity 0 is highest
		return nil
	}

	s := "<"+SeverityToString(severity)+"> "+message

	if fc.copyToStdOut && !strings.HasPrefix(message, "{") {
		fmt.Println(s)
	}

	return fc.writeMessage(s)
}

func (fc *FileClient) writeMessage(message string) error {
	f, err := os.OpenFile(fc.logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
	    panic(err)
	}

	defer f.Close()

	if _, err = f.WriteString(message); err != nil {
	    panic(err)
	}

	return nil
}