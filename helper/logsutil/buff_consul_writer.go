package logsutil

import (
	"fmt"
	"io"
	"path"
	"regexp"
	"time"

	"github.com/hashicorp/consul/api"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
)

type BufferedConsulWriter interface {
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

func NewBufferedConsulWriter(api *api.KV, deploymentID, prefix string) BufferedConsulWriter {
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
	err := consulutil.StoreConsulKey(path.Join(consulutil.DeploymentKVPrefix, b.deploymentID, "logs", b.prefix+"__"+time.Now().Format(time.RFC3339Nano)), out)
	if err != nil {
		return err
	}
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
