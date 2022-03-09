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

package kubernetes

import (
	"context"
	"fmt"
	"io/ioutil"
	"sort"
	"strings"
	"time"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/events"
	"github.com/ystia/yorc/v4/prov"
	"github.com/ystia/yorc/v4/prov/scheduling"
)

type jobLog struct {
	timestamp     time.Time
	podName       string
	containerName string
	line          string
}

type jobLogs []jobLog

func (l jobLogs) Len() int {
	return len(l)
}

func (l jobLogs) Less(i, j int) bool {
	return l[i].timestamp.Before(l[j].timestamp)
}

func (l jobLogs) Swap(i, j int) {
	l[i], l[j] = l[j], l[i]
}

func (l jobLogs) String() string {
	b := strings.Builder{}
	for i, jl := range l {
		if i != 0 {
			b.WriteString("\n")
		}
		b.WriteString(fmt.Sprintf("Pod: %q\tContainer: %q\t%s", jl.podName, jl.containerName, jl.line))
	}
	return b.String()
}

func publishJobLogs(ctx context.Context, cfg config.Configuration, clientset kubernetes.Interface, deploymentID, namespace, jobName string, action *prov.Action) error {
	podsList, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{LabelSelector: "job-name=" + jobName})
	if err != nil {
		return err
	}
	var since *time.Time
	sinceStr := action.Data["latestPublishedLogTimestamp"]
	if sinceStr != "" {
		t, err := time.Parse(time.RFC3339Nano, sinceStr)
		if err == nil {
			since = &t
		}
	}

	logs := make(jobLogs, 0)
	for _, pod := range podsList.Items {
		for _, container := range pod.Spec.Containers {
			l, err := getJobLogs(ctx, clientset, since, namespace, pod.Name, container.Name)
			if err != nil {
				return err
			}
			logs = append(logs, l...)
		}
	}
	sort.Sort(logs)

	if len(logs) > 0 && since != nil {
		// Timestamp is at the nanosec level while the log api is at the sec level
		// so we may have duplications here so let filter logs that are newer than
		// the registered timestamp
		b := logs[:0]
		for _, l := range logs {
			if l.timestamp.After(*since) {
				b = append(b, l)
			}
		}
		logs = b
	}

	if len(logs) > 0 {
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, deploymentID).Registerf("%s", logs)
		// store latest publish log timestamp
		lt := logs[len(logs)-1].timestamp
		cc, err := cfg.GetConsulClient()
		if err != nil {
			return err
		}
		scheduling.UpdateActionData(cc, action.ID, "latestPublishedLogTimestamp", lt.Format(time.RFC3339Nano))
	}
	return nil
}

func getJobLogs(ctx context.Context, clientset kubernetes.Interface, since *time.Time, namespace, podID, containerID string) ([]jobLog, error) {

	logOptions := &corev1.PodLogOptions{
		Container:  containerID,
		Follow:     false,
		Timestamps: true,
	}

	if since != nil {
		logOptions.SinceTime = &metav1.Time{Time: *since}
	}

	readCloser, err := clientset.CoreV1().Pods(namespace).GetLogs(podID, logOptions).Stream(ctx)

	if err != nil {
		return nil, errors.Wrapf(err, "failed to read logs for pod %q", podID)
	}
	defer readCloser.Close()
	b, err := ioutil.ReadAll(readCloser)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read logs for pod %q", podID)
	}

	return parseJobLogs(string(b), podID, containerID), errors.Wrapf(err, "failed to read logs for pod %q", podID)
}

func parseJobLogs(logs, podID, containerID string) []jobLog {
	lines := strings.Split(logs, "\n")
	r := make([]jobLog, 0, len(lines))
	for _, line := range lines {
		tokens := strings.SplitN(line, " ", 2)
		if len(tokens) < 2 {
			// not a timestamp + log so it is related to previous log
			if len(r) != 0 {
				r[len(r)-1].line += "\n" + line
			}
			continue
		}
		ts, err := time.Parse(time.RFC3339Nano, tokens[0])
		if err != nil {
			// not a timestamp + log so it is related to previous log
			if len(r) != 0 {
				r[len(r)-1].line += "\n" + line
			}
			continue
		}
		r = append(r, jobLog{timestamp: ts, line: tokens[1], podName: podID, containerName: containerID})
	}
	return r
}
