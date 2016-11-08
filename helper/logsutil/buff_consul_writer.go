package logsutil

import (
	"bytes"
	"fmt"
	"github.com/antonholmquist/jason"
	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"io"
	"novaforge.bull.com/starlings-janus/janus/deployments"
	"novaforge.bull.com/starlings-janus/janus/log"
	"path/filepath"
	"regexp"
	"time"
)

type BufferedWriter interface {
	Flush() error
	io.Writer
}

type BufferedConsulWriter interface {
	Run(quit chan bool)
	BufferedWriter
}

type ansibleJsonBufferedConsulWriter struct {
	kv          *api.KV
	depId       string
	buf         []byte
	jsonStarted bool
	prefix      string
}

type bufferedConsulWriter struct {
	kv     *api.KV
	depId  string
	buf    []byte
	prefix string
}

func NewBufferedConsulWriter(api *api.KV, depId, prefix string) BufferedConsulWriter {
	return &bufferedConsulWriter{
		buf:    make([]byte, 0),
		kv:     api,
		prefix: prefix,
		depId:  depId,
	}
}

func NewAnsibleJsonConsulWriter(api *api.KV, depId, prefix string) BufferedWriter {
	return &ansibleJsonBufferedConsulWriter{
		buf:    make([]byte, 0),
		kv:     api,
		depId:  depId,
		prefix: prefix,
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
	fmt.Printf(string(b.buf))
	reg := regexp.MustCompile(`\x1B\[([0-9]{1,2}(;[0-9]{1,2})?)?[m|K]`)
	out := reg.ReplaceAll(b.buf, []byte(""))
	kv := &api.KVPair{Key: filepath.Join(deployments.DeploymentKVPrefix, b.depId, "logs", b.prefix+"__"+time.Now().Format(time.RFC3339Nano)), Value: out}
	_, err := b.kv.Put(kv, nil)
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
				b.Flush()
				return
			case <-time.After(5 * time.Second):
				b.Flush()
			}
		}
	}()
}

func (b *ansibleJsonBufferedConsulWriter) Write(p []byte) (nn int, err error) {
	if !b.jsonStarted {
		// https://github.com/ansible/ansible/issues/17122
		if index := bytes.Index(p, []byte("{")); index >= 0 {
			log.Debugf("Skiped %d bytes from ansible logs", index)
			p = p[index:]
			b.jsonStarted = true
		} else {
			log.Debugf("Skiped all %d bytes from ansible logs", len(p))
			return 0, nil
		}
	}
	b.buf = append(b.buf, p...)
	return len(p), nil
}

//This function flush the buffer and write the content on Consul
func (b *ansibleJsonBufferedConsulWriter) Flush() error {

	//Construct the JSON from the buffer
	v, err := jason.NewObjectFromBytes(b.buf)
	if err != nil {
		err = errors.Wrap(err, "Ansible logs not available")
		log.Printf("%v", err)
		log.Debugf("%+v", err)
		log.Debugf("String: %q", string(b.buf))
		return err
	}

	//Get the array of object of plays
	plays, err := v.GetObjectArray("plays")
	for _, data := range plays {
		//Extract the tasks from the play
		tasks, err := data.GetObjectArray("tasks")
		if err != nil {
			err = errors.Wrap(err, "Ansible logs not available")
			log.Printf("%v", err)
			log.Debugf("%+v", err)
			continue
		}
		for _, host := range tasks {
			//Extract the hosts object from the  tasks
			tmp, err := host.GetObject("hosts")
			if err != nil {
				err = errors.Wrap(err, "Ansible logs not available")
				log.Printf("%v", err)
				log.Debugf("%+v", err)
				continue
			}
			//Convert the host into map like ["IP_ADDR"]Json_Object
			mapTmp := tmp.Map()
			//Iterate on this map (normally a single object)
			for k, v := range mapTmp {
				//Convert the value in Object type
				obj, err := v.Object()
				if err != nil {
					err = errors.Wrap(err, "Ansible logs not available")
					log.Printf("%v", err)
					log.Debugf("%+v", err)
					continue
				}
				//Check if a stderr field is present (The stdout field is exported for shell tasks on ansible)
				if std, err := obj.GetString("stderr"); err == nil && std != "" {
					//Display it and store it in consul
					//TODO: May interesting to store Host (IP Address) on consul, currently not done
					log.Debugf("Stderr found on host : %s  message : %s", k, std)
					kv := &api.KVPair{Key: filepath.Join(deployments.DeploymentKVPrefix, b.depId, "logs", b.prefix+"__"+time.Now().Format(time.RFC3339Nano)), Value: []byte("stderr: " + std)}
					_, err = b.kv.Put(kv, nil)
					if err != nil {
						err = errors.Wrap(err, "Ansible logs not available")
						log.Printf("%v", err)
						log.Debugf("%+v", err)
						continue
					}
				}
				//Check if a stdout field is present (The stdout field is exported for shell tasks on ansible)
				if std, err := obj.GetString("stdout"); err == nil && std != "" {
					//Display it and store it in consul
					//TODO: May interesting to store Host (IP Address) on consul, currently not done
					log.Debugf("Stdout found on host : %s  message : %s", k, std)
					kv := &api.KVPair{Key: filepath.Join(deployments.DeploymentKVPrefix, b.depId, "logs", b.prefix+"__"+time.Now().Format(time.RFC3339Nano)), Value: []byte("stdout: " + std)}
					_, err = b.kv.Put(kv, nil)
					if err != nil {
						err = errors.Wrap(err, "Ansible logs not available")
						log.Printf("%v", err)
						log.Debugf("%+v", err)
						continue
					}
				}

				//Check if a failed field is present
				if ok, err := obj.GetBoolean("failed"); ok {
					if err != nil {
						err = errors.Wrap(err, "Ansible logs not available")
						log.Printf("%v", err)
						log.Debugf("%+v", err)
						continue
					}
					//Get Error message
					str, err := obj.GetString("msg")
					if err != nil {
						err = errors.Wrap(err, "Ansible logs not available")
						log.Printf("%v", err)
						log.Debugf("%+v", err)
						continue
					}
					//Display it and store it in consul
					//TODO: May interesting to store Host (IP Address) on consul, currently not done
					log.Debugf("Error found on host : %s  message : %s", k, str)
					kv := &api.KVPair{Key: filepath.Join(deployments.DeploymentKVPrefix, b.depId, "logs", b.prefix+"__"+time.Now().Format(time.RFC3339Nano)), Value: []byte("msg: " + str)}
					_, err = b.kv.Put(kv, nil)
					if err != nil {
						err = errors.Wrap(err, "Ansible logs not available")
						log.Printf("%v", err)
						log.Debugf("%+v", err)
						continue
					}
				}
			}
		}

	}

	return nil
}
