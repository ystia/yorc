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

package ansible

import (
	"bytes"
	"context"
	"fmt"
	"strconv"

	"github.com/antonholmquist/jason"
	"github.com/pkg/errors"

	"github.com/ystia/yorc/events"
	"github.com/ystia/yorc/helper/collections"
	"github.com/ystia/yorc/log"
)

func getInstanceIDForHost(host string, hostConnectionMap map[string]hostConnection) string {
	instanceID := ""
	for _, connection := range hostConnectionMap {
		if host == connection.host {
			instanceID = connection.instanceID
			break
		}
	}
	return instanceID
}

func getAnsibleJSONResult(output *bytes.Buffer) (*jason.Object, []string, error) {
	// Workaround https://github.com/ansible/ansible/issues/17122
	log.Debugf("Ansible result: %s", output)
	b := output.Bytes()
	if i := bytes.Index(b, []byte("\"plays\":")); i >= 0 {
		b = append([]byte("{\n"), b[i:]...)
	} else {
		return nil, nil, errors.New("Not a valid JSON output")
	}
	//Construct the JSON from the buffer
	v, err := jason.NewObjectFromBytes(b)
	if err != nil {
		err = errors.Wrap(err, "Ansible logs not available")
		log.Printf("%v", err)
		log.Debugf("%+v", err)
		log.Debugf("String: %q", string(b))
		return nil, nil, err
	}

	failedHosts := make([]string, 0)
	stats, err := v.GetObject("stats")
	if err != nil {
		return nil, nil, errors.Wrap(err, "Failed to retrieve play stats")
	}
	for host, statsValue := range stats.Map() {
		statsObj, err := statsValue.Object()
		if err != nil {
			return nil, nil, errors.Wrap(err, "Failed to retrieve play stats")
		}
		failures, err := statsObj.GetInt64("failures")
		if err != nil {
			return nil, nil, errors.Wrap(err, "Failed to retrieve play stats")
		}
		if failures > 0 {
			failedHosts = append(failedHosts, host)
			continue
		}
		unreachables, err := statsObj.GetInt64("unreachable")
		if err != nil {
			return nil, nil, errors.Wrap(err, "Failed to retrieve play stats")
		}
		if unreachables > 0 {
			failedHosts = append(failedHosts, host)
		}
	}

	return v, failedHosts, nil
}

func checkAndPublishOutput(ctx context.Context, obj *jason.Object, deploymentID, nodeName, host, field string, logLevel events.LogLevel) {
	//Check if a msg field is present
	if std, err := obj.GetString(field); err == nil && std != "" {
		//Display it and store it in consul
		log.Debugf("%s found on host : %s  message : %s", field, host, std)
		events.WithContextOptionalFields(ctx).NewLogEntry(logLevel, deploymentID).RegisterAsString(fmt.Sprintf("node %q, host %q, %s:\n%s", nodeName, host, field, std))
	}
}

type logAnsibleOutputInConsulFn func(context.Context, string, string, map[string]hostConnection, *bytes.Buffer) error

func logAnsibleOutputInConsulFromScript(ctx context.Context, deploymentID, nodeName string, hostsConn map[string]hostConnection, output *bytes.Buffer) error {

	v, failedHosts, err := getAnsibleJSONResult(output)
	if err != nil {
		return err
	}

	//Get the array of object of plays
	plays, err := v.GetObjectArray("plays")
	if err != nil {
		return errors.Wrap(err, "Ansible logs not available")
	}
	for _, play := range plays {
		//Extract the tasks from the play
		tasks, err := play.GetObjectArray("tasks")
		if err != nil {
			err = errors.Wrap(err, "Ansible logs not available")
			log.Printf("%v", err)
			log.Debugf("%+v", err)
			continue
		}
		for _, task := range tasks {
			//Extract the hosts object from the  tasks
			tmp, err := task.GetObject("hosts")
			if err != nil {
				err = errors.Wrap(err, "Ansible logs not available")
				log.Printf("%v", err)
				log.Debugf("%+v", err)
				continue
			}
			//Convert the host into map like ["IP_ADDR"]Json_Object
			mapTmp := tmp.Map()
			//Iterate on this map (normally a single object)
			for host, v := range mapTmp {
				//Convert the value in Object type
				obj, err := v.Object()
				if err != nil {
					err = errors.Wrap(err, "Ansible logs not available")
					log.Printf("%v", err)
					log.Debugf("%+v", err)
					continue
				}
				stdErrLogLevel := events.LogLevelWARN
				logLevel := events.LogLevelINFO
				if collections.ContainsString(failedHosts, host) {
					stdErrLogLevel = events.LogLevelERROR
					logLevel = events.LogLevelERROR
				}
				lctx := events.AddLogOptionalFields(ctx, events.LogOptionalFields{events.InstanceID: getInstanceIDFromHosts(hostsConn, host)})
				checkAndPublishOutput(lctx, obj, deploymentID, nodeName, host, "module_stderr", stdErrLogLevel)
				checkAndPublishOutput(lctx, obj, deploymentID, nodeName, host, "module_stdout", logLevel)
				checkAndPublishOutput(lctx, obj, deploymentID, nodeName, host, "stderr", stdErrLogLevel)
				checkAndPublishOutput(lctx, obj, deploymentID, nodeName, host, "stdout", logLevel)
				checkAndPublishOutput(lctx, obj, deploymentID, nodeName, host, "msg", logLevel)
			}
		}

	}

	return nil
}

func appendAnsibleOutput(buf *bytes.Buffer, obj *jason.Object, field string) {
	if msg, err := obj.GetString(field); err == nil && msg != "" {
		buf.WriteString("\n\t")
		buf.WriteString(field)
		buf.WriteString(": \"")
		buf.WriteString(msg)
		buf.WriteString("\"")
	}
}

func logAnsibleOutputInConsul(ctx context.Context, deploymentID, nodeName string, hostsConn map[string]hostConnection, output *bytes.Buffer) error {

	v, failedHosts, err := getAnsibleJSONResult(output)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	//Get the array of object of plays
	plays, err := v.GetObjectArray("plays")
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve play name")
	}
	for i, play := range plays {
		var playName string
		playName, err = play.GetString("play", "name")
		if err != nil {
			return errors.Wrap(err, "Failed to retrieve play name")
		}
		if i != 0 {
			buf.WriteString("\n")
		}
		buf.WriteString("Ansible Playbook result:\n")
		buf.WriteString("\nPlay [")
		buf.WriteString(playName)
		buf.WriteString("]")
		log.Debugf("Play name is %q", playName)

		//Extract the tasks from the play
		var tasks []*jason.Object
		tasks, err = play.GetObjectArray("tasks")
		if err != nil {
			return errors.Wrap(err, "Failed to retrieve play tasks")
		}

		for _, task := range tasks {
			var taskName string
			taskName, err = task.GetString("task", "name")
			if err != nil {
				return errors.Wrap(err, "Failed to retrieve play tasks")
			}
			buf.WriteString("\n\nTask [")
			buf.WriteString(taskName)
			buf.WriteString("]")

			//Extract the hosts object from the  tasks
			var hosts *jason.Object
			hosts, err = task.GetObject("hosts")
			if err != nil {
				return errors.Wrap(err, "Failed to retrieve hosts results for play task")
			}

			//Iterate on this map (normally a single object)
			for hostName, hostVal := range hosts.Map() {
				buf.WriteString("\n")
				//Convert the value in Object type
				var host *jason.Object
				host, err = hostVal.Object()
				if err != nil {
					return errors.Wrapf(err, "Failed to retrieve task result for host %q", hostName)
				}
				var failed, unreachable, skipped, changed bool
				if failed, err = host.GetBoolean("failed"); err == nil && failed {
					buf.WriteString("failed: [")
				} else if unreachable, err = host.GetBoolean("unreachable"); err == nil && unreachable {
					buf.WriteString("unreachable: [")
				} else if skipped, err = host.GetBoolean("skipped"); err == nil && skipped {
					buf.WriteString("skipped: [")
				} else if changed, err = host.GetBoolean("changed"); err == nil && changed {
					buf.WriteString("changed: [")
				} else {
					buf.WriteString("ok: [")
				}
				buf.WriteString(hostName)
				buf.WriteString("]")
				buf.WriteString(" => {")
				braceLen := buf.Len()
				appendAnsibleOutput(&buf, host, "msg")
				appendAnsibleOutput(&buf, host, "stderr")
				appendAnsibleOutput(&buf, host, "stdout")
				results, err := host.GetObjectArray("results")
				if err == nil {
					// optionnal
					for _, r := range results {
						isItem, _ := r.GetBoolean("_ansible_item_result")
						if isItem {
							item, _ := r.GetString("item")
							if item != "" {
								buf.WriteString("\titem: \"")
								buf.WriteString(item)
								buf.WriteString("\"\n")
							}
						}
						if msg, err := r.GetString("msg"); err == nil && msg != "" {
							buf.WriteString("\tmsg: \"")
							buf.WriteString(msg)
							buf.WriteString("\"\n")
						}
						if sResults, err := r.GetStringArray("results"); err == nil && len(sResults) > 0 {
							buf.WriteString("\t{\n")
							for _, s := range sResults {
								buf.WriteString("\t\t")
								buf.WriteString(s)
								buf.WriteString("\n")
							}
							buf.WriteString("\t}\n")
						}
					}
				}
				if buf.Len() != braceLen {
					buf.WriteString("\n}")
				}
				buf.WriteString("}")
			}

		}
	}

	buf.WriteString("\nStats:")
	stats, err := v.GetObject("stats")
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve play stats")
	}
	for statsHost, statsValue := range stats.Map() {
		buf.WriteString("\nHost: ")
		buf.WriteString(statsHost)
		var statsObj *jason.Object
		statsObj, err = statsValue.Object()
		if err != nil {
			return errors.Wrap(err, "Failed to retrieve play stats")
		}
		changed, err := statsObj.GetInt64("changed")
		if err != nil {
			return errors.Wrap(err, "Failed to retrieve play stats")
		}
		buf.WriteString(" changed: ")
		buf.WriteString(strconv.FormatInt(changed, 10))
		failures, err := statsObj.GetInt64("failures")
		if err != nil {
			return errors.Wrap(err, "Failed to retrieve play stats")
		}
		buf.WriteString(" failures: ")
		buf.WriteString(strconv.FormatInt(failures, 10))
		ok, err := statsObj.GetInt64("ok")
		if err != nil {
			return errors.Wrap(err, "Failed to retrieve play stats")
		}
		buf.WriteString(" ok: ")
		buf.WriteString(strconv.FormatInt(ok, 10))
		skipped, err := statsObj.GetInt64("skipped")
		if err != nil {
			return errors.Wrap(err, "Failed to retrieve play stats")
		}
		buf.WriteString(" skipped: ")
		buf.WriteString(strconv.FormatInt(skipped, 10))
		unreachable, err := statsObj.GetInt64("unreachable")
		if err != nil {
			return errors.Wrap(err, "Failed to retrieve play stats")
		}
		buf.WriteString(" unreachable: ")
		buf.WriteString(strconv.FormatInt(unreachable, 10))

	}

	logLevel := events.LogLevelINFO
	if len(failedHosts) > 0 {
		logLevel = events.LogLevelERROR
	}

	// Register log entry
	events.WithContextOptionalFields(ctx).NewLogEntry(logLevel, deploymentID).Register(buf.Bytes())
	return nil
}

func getInstanceIDFromHosts(hosts map[string]hostConnection, hostname string) string {
	if hc, ok := hosts[hostname]; ok {
		return hc.instanceID
	}
	for _, hc := range hosts {
		if hc.host == hostname {
			return hc.instanceID
		}
	}
	return ""
}
