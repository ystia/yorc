package log

import (
	"fmt"
	"github.com/hashicorp/consul/api"
	"github.com/tidwall/gjson"
	"io"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)


const INFRA_LOG_PREFIX = "infrastructure"
const SOFTWARE_LOG_PREFIX = "software"
const ENGINE_LOG_PREFIX = "engine"
const ANSIBLE_OUTPUT_JSON_LOCATION = "plays.#.tasks.#.hosts.*.stdout"
const ANSIBLE_OUTPUT_JSON_FAIL_LOCATION = "plays.#.tasks.#.hosts.*.failed"
const ANSIBLE_OUTPUT_JSON_MSG_LOCATION = "plays.#.tasks.#.hosts.*.msg"

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

func (b *BufferedConsulWriter) FlushSoftware() error {
	if gjson.Get(string(b.buf), ANSIBLE_OUTPUT_JSON_FAIL_LOCATION).Exists() {
		out := gjson.Get(string(b.buf), ANSIBLE_OUTPUT_JSON_MSG_LOCATION).String()
		out = strings.TrimPrefix(out, "[[,")
		out = strings.TrimSuffix(out, "]]")
		out, err := strconv.Unquote(out)
		Debugf(out)
		if err != nil {
			return err
		}
		kv := &api.KVPair{Key: filepath.Join(b.prefix, b.depId, "logs", SOFTWARE_LOG_PREFIX+"__"+time.Now().Format(time.RFC3339Nano)), Value: []byte(out)}
		_, err = b.kv.Put(kv, nil)
		if err != nil {
			return err
		}
	}
	if gjson.Get(string(b.buf), ANSIBLE_OUTPUT_JSON_LOCATION).Exists() {
		out := gjson.Get(string(b.buf), ANSIBLE_OUTPUT_JSON_LOCATION).String()
		out = strings.TrimPrefix(out, "[[,")
		out = strings.TrimSuffix(out, "]]")
		out, err := strconv.Unquote(out)
		Debugf(out)
		if err != nil {
			return err
		}
		kv := &api.KVPair{Key: filepath.Join(b.prefix, b.depId, "logs", SOFTWARE_LOG_PREFIX+"__"+time.Now().Format(time.RFC3339Nano)), Value: []byte(out)}
		_, err = b.kv.Put(kv, nil)
		if err != nil {
			return err
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
