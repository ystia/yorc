// Copyright 2018 Bull S.A.S. Atos Technologies - Bull, Rue Jean Jaures, B.P.68, 78340, Les Clayes-sous-Bois, France.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package events

import (
	"io"
	"regexp"
	"time"

	"github.com/ystia/yorc/log"
)

// A BufferedLogEntryWriter is a Writer that buffers writes and flushes its buffer as an event log on a regular basis (every 5s)
type BufferedLogEntryWriter interface {
	run(quit chan bool, logEntry LogEntry)
	flush(logEntry LogEntry) error
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
func (b *bufferedConsulWriter) flush(logEntry LogEntry) error {
	if len(b.buf) == 0 {
		return nil
	}
	log.Debug(string(b.buf))
	reg := regexp.MustCompile(`\x1B\[([0-9]{1,2}(;[0-9]{1,2})?)?[m|K]`)
	out := reg.ReplaceAll(b.buf, []byte(""))
	logEntry.Register(out)
	b.buf = b.buf[:0]
	return nil
}

// Internal : run allows to run buffering and allows flushing when done or after timeout
func (b *bufferedConsulWriter) run(quit chan bool, logEntry LogEntry) {
	go func() {
		for {
			select {
			case <-quit:
				err := b.flush(logEntry)
				if err != nil {
					log.Print(err)
				}
				return
			case <-time.After(b.timeout):
				err := b.flush(logEntry)
				if err != nil {
					log.Print(err)
				}
			}
		}
	}()
}
