package log

import (
	"fmt"
	"github.com/hashicorp/consul/api"
	"io"
	"path/filepath"
	"regexp"
	"time"
	"github.com/antonholmquist/jason"
)


const INFRA_LOG_PREFIX = "infrastructure"
const SOFTWARE_LOG_PREFIX = "software"
const ENGINE_LOG_PREFIX = "engine"

type BufferedConsulWriter struct {
	kv     *api.KV
	depId  string
	buf    []byte
	prefix string
	io.Writer
}

func NewWriterSize(api *api.KV, depId string, prefix string) *BufferedConsulWriter {
	return &BufferedConsulWriter{
		buf:    make([]byte, 0),
		kv:     api,
		prefix: prefix,
		depId:  depId,
	}
}

func (b *BufferedConsulWriter) Write(p []byte) (nn int, err error) {
	b.buf = append(b.buf, p...)
	return len(p), nil
}

func (b *BufferedConsulWriter) Flush() error {
	if len(b.buf) == 0 {
		return nil
	}
	fmt.Printf(string(b.buf))
	reg := regexp.MustCompile(`\x1B\[([0-9]{1,2}(;[0-9]{1,2})?)?[m|K]`)
	out := reg.ReplaceAll(b.buf, []byte(""))
	kv := &api.KVPair{Key: filepath.Join(b.prefix, b.depId, "logs", INFRA_LOG_PREFIX+"__"+time.Now().Format(time.RFC3339Nano)), Value: out}
	_, err := b.kv.Put(kv, nil)
	if err != nil {
		return err
	}
	b.buf = b.buf[:0]
	return nil

}

//Simple function to check possible Error
func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

//This function flush the buffer and write the content on Consul
func (b *BufferedConsulWriter) FlushSoftware() error {
	v, err := jason.NewObjectFromBytes(b.buf)
	if err != nil {
		panic(err)
	}

	plays, err := v.GetObjectArray("plays")
	for _, data := range plays {
		tasks, err := data.GetObjectArray("tasks")
		checkErr(err)
		for _, host := range tasks{
			tmp, err := host.GetObject("hosts")
			checkErr(err)
			mapTmp := tmp.Map()
			for k, v := range mapTmp{
				tmp2, err := v.Object()
				checkErr(err)
				if ok, err := tmp2.GetBoolean("failed"); ok {
					checkErr(err)
					str, err := tmp2.GetString("msg")
					checkErr(err)
					Debugf("Error found on host : %s  message : %s",k, str)
					kv := &api.KVPair{Key: filepath.Join(b.prefix, b.depId, "logs", SOFTWARE_LOG_PREFIX+"__"+time.Now().Format(time.RFC3339Nano)), Value: []byte(str)}
					_, err = b.kv.Put(kv, nil)
					checkErr(err)
				}
				if std, err := tmp2.GetString("stdout"); err == nil {
					Debugf("Stdout found on host : %s  message : %s",k, std)
					kv := &api.KVPair{Key: filepath.Join(b.prefix, b.depId, "logs", SOFTWARE_LOG_PREFIX+"__"+time.Now().Format(time.RFC3339Nano)), Value: []byte(std)}
					_, err = b.kv.Put(kv, nil)
					checkErr(err)
				}
			}
		}

	}

	return nil
}

func (b *BufferedConsulWriter) Run(quit chan bool) {
	go func() {
		for {
			select {
			case <-quit:
				return
			case <-time.After(5 * time.Second):
				b.Flush()
			}
		}
	}()
}
