package log

import (
	"io"
	slog "log"
	"os"
	"strings"
	"sync"
)

var (
	std = slog.New(os.Stdout, "", slog.LstdFlags)
	debug = false
	mutex sync.Mutex
)

func init() {
	switch strings.ToUpper(os.Getenv("JANUS_LOG")) {
	case "DEBUG":
		debug = true
	case "1":
		debug = true
	}
}

func SetDebug(d bool) {
	mutex.Lock()
	defer mutex.Unlock()
	debug = d
}

// SetOutput sets the output destination for the standard logger.
func SetOutput(w io.Writer) {
	std.SetOutput(w)
}

// Flags returns the output flags for the standard logger.
func Flags() int {
	return std.Flags()
}

// SetFlags sets the output flags for the standard logger.
func SetFlags(flag int) {
	std.SetFlags(flag)
}

// Prefix returns the output prefix for the standard logger.
func Prefix() string {
	return std.Prefix()
}

// SetPrefix sets the output prefix for the standard logger.
func SetPrefix(prefix string) {
	std.SetPrefix(prefix)
}

// These functions write to the standard logger.

// Print calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Print.
func Print(v ...interface{}) {
	a := make([]interface{}, len(v) + 1)
	a[0] = "[INFO] "
	std.Print(append(a, v...))
}

// Printf calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func Printf(format string, v ...interface{}) {
	std.Printf("[INFO]  " + format, v...)
}

// Println calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Println.
func Println(v ...interface{}) {
	a := make([]interface{}, len(v) + 1)
	a[0] = "[INFO] "
	std.Println(append(a, v...))
}

// Fatal is equivalent to Print() followed by a call to os.Exit(1).
func Fatal(v ...interface{}) {
	a := make([]interface{}, len(v) + 1)
	a[0] = "[FATAL]"
	std.Fatal(append(a, v...))
}

// Fatalf is equivalent to Printf() followed by a call to os.Exit(1).
func Fatalf(format string, v ...interface{}) {
	std.Fatalf("[FATAL] " + format, v...)
}

// Fatalln is equivalent to Println() followed by a call to os.Exit(1).
func Fatalln(v ...interface{}) {
	a := make([]interface{}, len(v) + 1)
	a[0] = "[FATAL]"
	std.Fatalln(append(a, v...))
}

// Panic is equivalent to Print() followed by a call to panic().
func Panic(v ...interface{}) {
	a := make([]interface{}, len(v) + 1)
	a[0] = "[PANIC]"
	std.Panic(append(a, v...))
}

// Panicf is equivalent to Printf() followed by a call to panic().
func Panicf(format string, v ...interface{}) {
	std.Printf("[PANIC] " + format, v...)
}

// Panicln is equivalent to Println() followed by a call to panic().
func Panicln(v ...interface{}) {
	a := make([]interface{}, len(v) + 1)
	a[0] = "[PANIC]"
	std.Panicln(append(a, v...))
}

// Output writes the output for a logging event.  The string s contains
// the text to print after the prefix specified by the flags of the
// Logger.  A newline is appended if the last character of s is not
// already a newline.  Calldepth is the count of the number of
// frames to skip when computing the file name and line number
// if Llongfile or Lshortfile is set; a value of 1 will print the details
// for the caller of Output.
func Output(calldepth int, s string) error {
	return std.Output(calldepth + 1, "[INFO] " + s) // +1 for this frame.
}

// Print calls Output to print to the standard logger if debug is enable.
// Arguments are handled in the manner of fmt.Print.
func Debug(v ...interface{}) {
	if debug {
		a := make([]interface{}, len(v) + 1)
		a[0] = "[DEBUG]"
		std.Print(append(a, v...))
	}

}

// Printf calls Output to print to the standard logger if debug is enable.
// Arguments are handled in the manner of fmt.Printf.
func Debugf(format string, v ...interface{}) {
	if debug {
		std.Printf("[DEBUG] " + format, v...)
	}
}

// Println calls Output to print to the standard logger if debug is enable.
// Arguments are handled in the manner of fmt.Println.
func Debugln(v ...interface{}) {
	if debug {
		a := make([]interface{}, len(v) + 1)
		a[0] = "[DEBUG]"
		std.Println(append(a, v...))
	}
}
