package log

import (
	"fmt"
	"time"
    "encoding/json"
)

type LogInterface interface {
    WriteMessage(message string) error
}

var logEngine LogInterface

func SetUpNetworkLogEngine(remoteHost string, copyToStdout bool) {
    logEngine = StartSinkClient(remoteHost, copyToStdout)
}

func SetUpFileLogEngine(logFile string, copyToStdout bool) {
    logEngine = StartFileClient(logFile, copyToStdout)
}

/*
 * Aux methods
 */

func SimpleStringDump(s string) {
    logEngine.WriteMessage(s)
}

func JsonDump(data interface{}) {
	b, err := json.Marshal(data)
    if err != nil {
        fmt.Println(err)
        return
    }
    s := string(b)

    logEngine.WriteMessage(s)
}

func BenchmarkInt(experiment string, duration int) {
    when := time.Now()
	s := fmt.Sprintf("{\"time\":\"", when, "\", \"experiment\":\"%q\", \"time\":%d}", experiment, duration)
	logEngine.WriteMessage(s)
}

func BenchmarkFloat(experiment string, duration float64) {
    when := time.Now()
	s := fmt.Sprintf("{\"time\":\"", when, "\", \"experiment\":\"%q\", \"time\":%f}", experiment, duration)
	logEngine.WriteMessage(s)
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
    when := time.Now()
	s := fmt.Sprint("[", when, "] Entity ", entity, " did ", task, " in ", duration, "\n")
    logEngine.WriteMessage(s)

    s2 := fmt.Sprint("{\"time\":\"", when, "\", \"entity\":\"", entity, "\", \"task\":\"", task, "\", \"duration\":\"", duration, "\"}")
    logEngine.WriteMessage(s2)
}

func InfoReport(entity, info string) {
    when := time.Now()
    s := fmt.Sprint("[", when, "] Entity ", entity, " did ", info, "\n")
    logEngine.WriteMessage(s)

    s2 := fmt.Sprint("{\"time\":\"", when, "\", \"entity\":\"", entity, "\", \"info\":\"", info, "\"}")
    logEngine.WriteMessage(s2)
}