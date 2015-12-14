package log

import (
	"fmt"
	"time"
    "encoding/binary"
    "encoding/json"
)

const (
    SEVERE_ERROR = iota
    RECOVERABLE_ERROR
    EXPERIMENT_OUTPUT
    WARNING
    NOTIFICATION
    INFORMATION
)

func SeverityToString(severity int) string {
    switch(severity) {
        case SEVERE_ERROR:
            return "ERR0"
            break
        case RECOVERABLE_ERROR:
            return "ERR1"
            break
        case EXPERIMENT_OUTPUT:
            return "EXPE"
            break
        case WARNING:
            return "WARN"
            break
        case NOTIFICATION:
            return "NOTI"
            break
        case INFORMATION:
            return "INFO"
            break        
    }
    return "UNKN"
}

type LogInterface interface {
    WriteMessage(severity int, message string) error
}

var logEngine LogInterface

func SetUpNetworkLogEngine(logLevel int, entity string, remoteHost string, copyToStdout bool) {
    logEngine = StartSinkClient(logLevel, entity, remoteHost, copyToStdout)
}

func SetUpFileLogEngine(logLevel int, logFile string, copyToStdout bool) {
    logEngine = StartFileClient(logLevel, logFile, copyToStdout)
}

/*
 * Aux methods
 */

func Println(severity int, a ...interface{}) {
    s := fmt.Sprint(a)
    SimpleStringDump(severity, s)
}

func Printf(severity int, format string, a ...interface{}) {
    s := fmt.Sprintf(format, a...)
    SimpleStringDump(severity, s)
}

func SimpleStringDump(severity int, s string) {
    logEngine.WriteMessage(severity, s)
}

func JsonDump(data interface{}) {
	b, err := json.Marshal(data)
    if err != nil {
        fmt.Println(err)
        return
    }
    s := string(b)

    logEngine.WriteMessage(EXPERIMENT_OUTPUT, s)
}

func BenchmarkInt(experiment string, duration int) {
    when := time.Now().Format(time.StampMilli)
	s := fmt.Sprintf("{\"time\":\"", when, "\", \"experiment\":\"%q\", \"time\":%d}", experiment, duration)
	logEngine.WriteMessage(EXPERIMENT_OUTPUT, s)
}

func BenchmarkFloat(experiment string, duration float64) {
    when := time.Now().Format(time.StampMilli)
	s := fmt.Sprintf("{\"time\":\"", when, "\", \"experiment\":\"%q\", \"time\":%f}", experiment, duration)
	logEngine.WriteMessage(EXPERIMENT_OUTPUT, s)
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
    when := time.Now().Format(time.StampMilli)
	s := fmt.Sprint("[", when, "] Entity ", entity, " did ", task, " in ", duration)
    logEngine.WriteMessage(EXPERIMENT_OUTPUT, s)

    s2 := fmt.Sprint("{\"time\":\"", when, "\", \"entity\":\"", entity, "\", \"task\":\"", task, "\", \"duration\":\"", duration, "\"}")
    logEngine.WriteMessage(EXPERIMENT_OUTPUT, s2)
}

func InfoReport(severity int, entity, info string) {
    when := time.Now().Format(time.StampMilli)
    s := fmt.Sprint("[", when, "] Entity ", entity, " did ", info)
    logEngine.WriteMessage(severity, s)

    s2 := fmt.Sprint("{\"time\":\"", when, "\", \"entity\":\"", entity, "\", \"info\":\"", info, "\"}")
    logEngine.WriteMessage(severity, s2)
}

func MsTimeStamp() int64 {
    //http://stackoverflow.com/questions/24122821/go-golang-time-now-unixnano-convert-to-milliseconds
    return time.Now().UnixNano() / int64(time.Millisecond)
}

func MsTimeStampString() string {
    currTime := MsTimeStamp()
    return fmt.Sprintf("%v", currTime)
}

func MsTimeStampBytes() []byte {
    currTime := MsTimeStamp()
    buf := make([]byte, 8)
    binary.BigEndian.PutUint64(buf[0:8], uint64(currTime))
    return buf
}