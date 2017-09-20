package main

import (
	"fmt"
	"net/http"
	"os/exec"
	"strings"
)

var header = "<html><head><style>h1{ margin-bottom:30px; } a {display:block;}</style></head><body>"
var footer = "</body></html>"

func index(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, header)
	fmt.Fprintf(w, "<h1>PriFi</h1><a href=\"/reboot\">Hard-reset the PriFi protocol</a>")

	stdout, stderr := exec.Command("./list-prifi.sh").Output()

	if stderr != nil {
		fmt.Fprintf(w, "<p>%s</p>", string(stderr.Error()))
	} else {
		s := string(stdout)
		s = strings.Replace(s, "\n", "<br>", -1)
		fmt.Fprintf(w, "<p>%s</p>", s)
	}
	fmt.Fprintf(w, footer)
}

func reboot(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, header)
	fmt.Fprintf(w, "<h1>PriFi</h1>")
	stdout, stderr := exec.Command("./rerun.sh").Output()

	if stderr != nil {
		fmt.Fprintf(w, "<p>%s</p>", string(stderr.Error()))
	} else {
		fmt.Fprintf(w, "<p>%s</p>", string(stdout))
	}

	fmt.Fprintf(w, "<a href=\"/\">Back</a>")
	fmt.Fprintf(w, footer)
}

func main() {
	http.HandleFunc("/", index)
	http.HandleFunc("/reboot", reboot)
	http.ListenAndServe(":8080", nil)
}
