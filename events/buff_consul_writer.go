package events

import (
	"fmt"
	"io"
	"regexp"
	"time"

	"github.com/hashicorp/consul/api"
)

// A BufferedLogEventWriter is a Writer that buffers writes and flushes its buffer as an event log on a regular basis (every 5s)
type BufferedLogEventWriter interface {
	Run(quit chan bool)
	Flush() error
	io.Writer
}

type bufferedConsulWriter struct {
	kv           *api.KV
	deploymentID string
	buf          []byte
	prefix       string
}

// NewBufferedLogEventWriter returns a BufferedLogEventWriter that will use prefix to publish logs
func NewBufferedLogEventWriter(api *api.KV, deploymentID, prefix string) BufferedLogEventWriter {
	return &bufferedConsulWriter{
		buf:          make([]byte, 0),
		kv:           api,
		prefix:       prefix,
		deploymentID: deploymentID,
	}
}

func (b *bufferedConsulWriter) Write(p []byte) (nn int, err error) {
	b.buf = append(b.buf, p...)
	return len(p), nil
}

func (b *bufferedConsulWriter) Flush() error {
	if len(b.buf) == 0 {
		return nil
	}
	fmt.Print(string(b.buf))
	reg := regexp.MustCompile(`\x1B\[([0-9]{1,2}(;[0-9]{1,2})?)?[m|K]`)
	out := reg.ReplaceAll(b.buf, []byte(""))
	logInConsul(b.kv, b.deploymentID, b.prefix, out)
	b.buf = b.buf[:0]
	return nil

}

func (b *bufferedConsulWriter) Run(quit chan bool) {
	go func() {
		for {
			select {
			case <-quit:
				err := b.Flush()
				if err != nil {
					fmt.Print(err)
				}
				return
			case <-time.After(5 * time.Second):
				err := b.Flush()
				if err != nil {
					fmt.Print(err)
				}
			}
		}
	}()
}
