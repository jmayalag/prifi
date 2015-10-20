package log

import (
	"os"
	"fmt"
	"log"
    "encoding/json"
)

var logFile = "dissent.log"
var entity = ""

func StringDump(s string) {
    writeToLogFile(s)
}

func JsonDump(data interface{}) {
	b, err := json.Marshal(data)
    if err != nil {
        fmt.Println(err)
        return
    }
    s := string(b)

    writeToLogFile(s)
}

func BenchmarkInt(experiment string, duration int) {
	s := fmt.Sprintf("{\"experiment\":\"%q\", \"time\":%d}", experiment, duration)
	writeToLogFile(s)
}

func BenchmarkFloat(experiment string, duration float64) {
	s := fmt.Sprintf("{\"experiment\":\"%q\", \"time\":%f}", experiment, duration)
	writeToLogFile(s)
}

func writeToLogFile(s string) {
	f, err := os.OpenFile(logFile, os.O_RDWR | os.O_CREATE | os.O_APPEND, 0666)
	if err != nil {
	    panic("log : error opening file.")
	}
	defer f.Close()

	log.SetOutput(f)
	log.Println(s)
}
