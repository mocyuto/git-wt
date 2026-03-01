package logger

import (
	"fmt"
	"os"
	"strings"
)

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
)

var Verbose bool

func Error(format string, a ...interface{}) {
	fmt.Fprint(os.Stderr, Errorf(format, a...).Error()+"\n")
}

func Errorf(format string, a ...interface{}) error {
	msg := fmt.Sprintf(format, a...)
	if strings.Contains(msg, colorRed) {
		return fmt.Errorf("%s", msg)
	}
	return fmt.Errorf(colorRed+"Error: "+format+colorReset, a...)
}

func Warn(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, colorYellow+"Warning: "+format+colorReset+"\n", a...)
}

func Success(format string, a ...interface{}) {
	fmt.Printf(colorGreen+format+colorReset+"\n", a...)
}

func Info(format string, a ...interface{}) {
	fmt.Printf(colorCyan+format+colorReset+"\n", a...)
}

func Log(format string, a ...interface{}) {
	fmt.Printf(format+"\n", a...)
}
