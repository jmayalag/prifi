package log

import (
	"os"
	"fmt"
	"log"
	"time"
    "encoding/json"
	"github.com/fatih/color"
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

/* Usage :
func factorial(n *big.Int) (result *big.Int) {
    defer timeTrack(time.Now(), "factorial")
    // ... do some things, maybe even return under some condition
    return n
}
*/
func TimeTrack(entity, task string, start time.Time) {
    elapsed := time.Since(start)
    StatisticReport(entity, task, elapsed.String())
}

func StatisticReport(entity, task, duration string) {
	s := fmt.Sprint("[Timings] Entity %s did %s in %s\n", entity, task, duration)
	color.White(s)
    writeToLogFile(s)
    log.Printf(s)

    s2 := fmt.Sprintf("{\"entity\":\"%q\", \"task\":\"%q\", \"time\":%d}", entity, task, duration)
    writeToLogFile(s2)
}