package events

import (
	"fmt"
	"io"
	"regexp"
	"time"
)

// A BufferedLogEntryWriter is a Writer that buffers writes and flushes its buffer as an event log on a regular basis (every 5s)
type BufferedLogEntryWriter interface {
	run(quit chan bool, logEntry FormattedLogEntry)
	flush(logEntry FormattedLogEntry) error
	io.Writer
}

// bufferedConsulWriter is internal BufferedLogEntryWriter implementation
type bufferedConsulWriter struct {
	buf     []byte
	timeout time.Duration
}

// NewBufferedLogEntryWriter returns a BufferedLogEntryWriter used to register log entry
func NewBufferedLogEntryWriter() BufferedLogEntryWriter {
	return &bufferedConsulWriter{
		buf:     make([]byte, 0),
		timeout: 5 * time.Second,
	}
}

// Write allows to write bytes into a bufferedConsulWriter
func (b *bufferedConsulWriter) Write(p []byte) (nn int, err error) {
	b.buf = append(b.buf, p...)
	return len(p), nil
}

// Internal : flush allows to flush buffer content into Consul
func (b *bufferedConsulWriter) flush(logEntry FormattedLogEntry) error {
	if len(b.buf) == 0 {
		return nil
	}
	fmt.Print(string(b.buf))
	reg := regexp.MustCompile(`\x1B\[([0-9]{1,2}(;[0-9]{1,2})?)?[m|K]`)
	out := reg.ReplaceAll(b.buf, []byte(""))
	logEntry.Register(out)
	b.buf = b.buf[:0]
	return nil
}

// Internal : run allows to run buffering and allows flushing when done or after timeout
func (b *bufferedConsulWriter) run(quit chan bool, logEntry FormattedLogEntry) {
	go func() {
		for {
			select {
			case <-quit:
				err := b.flush(logEntry)
				if err != nil {
					fmt.Print(err)
				}
				return
			case <-time.After(b.timeout):
				err := b.flush(logEntry)
				if err != nil {
					fmt.Print(err)
				}
			}
		}
	}()
}
