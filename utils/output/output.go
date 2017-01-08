// Package output defines an interface to write
// text messages. It provides implementations
// of this interface that use standard output,
// Cothority's logging infrastructure to write
// the messages or simply discards them.
package output

import (
	"fmt"
	"github.com/dedis/cothority/log"
)

// Output Interface provides a Print function.
type Output interface {
	Print(text string)
}

// PrintOutput prints it's messages to the standard output.
type PrintOutput struct {}

func(o PrintOutput) Print(text string) {
	fmt.Println(text)
}

// LogOutput prints it's messages using Cothority's logging infrastructure.
type LogOutput struct {
	Level int
	Info  bool
}

func(o LogOutput) Print(text string) {
	if (o.Info) {
		log.Info(text)
	} else {
		switch o.Level {
		case 1: log.Lvl1(text)
		case 2: log.Lvl2(text)
		case 3: log.Lvl3(text)
		case 4: log.Lvl4(text)
		case 5: log.Lvl5(text)
		default:
			log.Print(text)
		}
	}
}

// NullOutput prints discards all messages.
type NullOutput struct {}

func(o NullOutput) Print(text string) {}
