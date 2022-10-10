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

package workflow

import (
	"context"
	"fmt"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"

	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/log"
	"github.com/ystia/yorc/v4/tasks"
)

// A taskExecution is the unit task of execution. If task is a workflow, it contains as many TaskExecutions as workflow steps
type taskExecution struct {
	id           string
	taskID       string
	targetID     string
	taskType     tasks.TaskType
	creationDate time.Time
	lock         *api.Lock
	cc           *api.Client
	step         string
	// finalFunction is function a function called at the end of the taskExecution if no other taskExecution are running
	finalFunction func() error
}

func (t *taskExecution) releaseLock() {
	if t.lock != nil {
		t.lock.Unlock()
		t.lock.Destroy()
	}
}

func acquireRunningExecLock(cc *api.Client, taskID string) (*consulutil.AutoDeleteLock, error) {
	execPath := path.Join(consulutil.TasksPrefix, taskID, ".runningExecutionsLock")
RETRY:
	execLock, err := cc.LockKey(execPath)
	if err != nil {
		return nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}

	lCh, err := execLock.Lock(nil)
	if err != nil {
		return nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if lCh == nil {
		goto RETRY
	}
	return &consulutil.AutoDeleteLock{Lock: execLock}, nil

}

func (t *taskExecution) notifyStart() error {
	execLock, err := acquireRunningExecLock(t.cc, t.taskID)
	if err != nil {
		return err
	}
	defer execLock.Unlock()

	consulNodeName, err := t.cc.Agent().NodeName()
	if err != nil {
		return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	log.Debugf("Storing runningExecutions with id %q for task %q", t.id, t.taskID)
	return consulutil.StoreConsulKeyAsString(path.Join(consulutil.TasksPrefix, t.taskID, ".runningExecutions", t.id), consulNodeName)
}

func numberOfWaitingExecutionsForTask(cc *api.Client, taskID string) (int, error) {
	var nbWaiting int
	l, err := acquireRunningExecLock(cc, taskID)
	if err != nil {
		return 0, err
	}
	defer l.Unlock()
	execPath := path.Join(consulutil.TasksPrefix, taskID, ".runningExecutions")
	kv := cc.KV()
	keys, _, err := kv.Keys(execPath+"/", "/", nil)
	if err != nil {
		return 0, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	for _, execKey := range keys {
		execID := path.Base(execKey)
		if execWaiting, err := isWaitingExec(taskID, execID); err == nil && execWaiting {
			nbWaiting++
		}
	}
	return nbWaiting, nil
}

func isWaitingExec(taskID, execID string) (bool, error) {
	exist, step, err := consulutil.GetStringValue(path.Join(consulutil.ExecutionsTaskPrefix, execID, "step"))
	if err != nil || !exist {
		return false, errors.Errorf("Cannot get the step of execution %q", execID)
	}
	status, err := tasks.GetTaskStepStatus(taskID, step)
	if err != nil {
		return false, errors.Errorf("Cannot get status for step %q in task %q", step, taskID)
	}
	if status == tasks.TaskStepStatusINITIAL {
		return true, nil
	}
	return false, nil
}

func numberOfRunningExecutionsForTask(cc *api.Client, taskID string) (*consulutil.AutoDeleteLock, int, error) {
	l, err := acquireRunningExecLock(cc, taskID)
	if err != nil {
		return nil, 0, err
	}
	execPath := path.Join(consulutil.TasksPrefix, taskID, ".runningExecutions")
	kv := cc.KV()
	// Check if we were the latest
	keys, _, err := kv.Keys(execPath+"/", "/", nil)
	if err != nil {
		l.Unlock()
		return nil, 0, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	log.Debugf("numberOfRunningExecutionsForTask %q: %d", taskID, len(keys))
	return l, len(keys), nil
}

func (t *taskExecution) notifyEnd() error {
	log.Debugf("notifyEnd for taskExecution %q", t.id)
	execPath := path.Join(consulutil.TasksPrefix, t.taskID, ".runningExecutions")
	registeredPath := path.Join(consulutil.TasksPrefix, t.taskID, ".registeredExecutions")
	l, e, err := numberOfRunningExecutionsForTask(t.cc, t.taskID)
	if err != nil {
		return err
	}
	defer l.Unlock()

	kv := t.cc.KV()
	// Delete our execID
	ops := api.KVTxnOps{
		&api.KVTxnOp{
			Verb: api.KVDelete,
			Key:  path.Join(execPath, t.id),
		},
		&api.KVTxnOp{
			Verb: api.KVDelete,
			Key:  path.Join(registeredPath, t.step),
		},
	}
	log.Debugf("Deleting runningExecutions with id %q for task %q", t.id, t.taskID)
	ok, response, _, err := kv.Txn(ops, nil)
	if err != nil {
		return errors.Wrap(err, "Failed to execute transaction")
	}
	if !ok {
		// Check the response
		var errs []string
		for _, e := range response.Errors {
			errs = append(errs, e.What)
		}
		return errors.Errorf("Failed to execute transaction: %s", strings.Join(errs, ", "))
	}
	if e <= 1 && t.finalFunction != nil {
		log.Debugf("notifyEnd running finalFunction for taskExecution %q", t.id)
		return t.finalFunction()
	}
	return nil
}

func (t *taskExecution) delete() error {
	_, err := t.cc.KV().DeleteTree(path.Join(consulutil.ExecutionsTaskPrefix, t.id)+"/", nil)
	return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
}

func (t *taskExecution) getTaskStatus() (tasks.TaskStatus, error) {
	return tasks.GetTaskStatus(t.taskID)
}

// checkAndSetTaskStatus allows to check the task status before updating it
func checkAndSetTaskStatus(ctx context.Context, targetID, taskID string, finalStatus tasks.TaskStatus, errReason error) error {
	kvp, meta, err := consulutil.GetKV().Get(path.Join(consulutil.TasksPrefix, taskID, "status"), nil)
	if err != nil {
		return errors.Wrapf(err, "Failed to get task status for taskID:%q", taskID)
	}
	if kvp == nil || len(kvp.Value) == 0 {
		return errors.Wrapf(err, "Missing task status for taskID:%q", taskID)
	}
	st, err := strconv.Atoi(string(kvp.Value))
	if err != nil {
		return errors.Wrapf(err, "Invalid task status for taskID:%q", taskID)
	}

	status := tasks.TaskStatus(st)
	// TaskStatusFAILED and TaskStatusCANCELED are terminal status and can't be changed
	if finalStatus != status {
		if status == tasks.TaskStatusFAILED {
			mess := fmt.Sprintf("Can't set task status with taskID:%q to:%q because task status is FAILED", taskID, finalStatus.String())
			log.Printf(mess)
			return errors.Errorf(mess)
		} else if status == tasks.TaskStatusCANCELED {
			mess := fmt.Sprintf("Can't set task status with taskID:%q to:%q because task status is CANCELED", taskID, finalStatus.String())
			log.Printf(mess)
			return errors.Errorf(mess)
		}
		return setTaskStatus(ctx, targetID, taskID, finalStatus, meta.LastIndex, errReason)
	}
	return nil
}

func setTaskStatus(ctx context.Context, targetID, taskID string, status tasks.TaskStatus, lastIndex uint64, errReason error) error {
	p := &api.KVPair{Key: path.Join(consulutil.TasksPrefix, taskID, "status"), Value: []byte(strconv.Itoa(int(status)))}
	p.ModifyIndex = lastIndex
	set, _, err := consulutil.GetKV().CAS(p, nil)
	if err != nil {
		log.Printf("Failed to set status to %q for taskID:%q due to error:%+v", status.String(), taskID, err)
		return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if !set {
		log.Debugf("[WARNING] Failed to set task status to:%q for taskID:%q as last index has been changed before. Retry it", status.String(), taskID)
		return checkAndSetTaskStatus(ctx, targetID, taskID, status, errReason)
	}

	// Emit event for status change
	// wfName may be empty as this data is not filled for non-workflow task type (as for custom command by instance)
	wfName, _ := tasks.GetTaskData(taskID, "workflowName")
	taskType, err := tasks.GetTaskType(taskID)
	// As task has been set, error are ignored but taskType is mandatory for emitting event
	if err != nil {
		log.Printf("[WARNING] Failed to emit event for change status to %q for taskID:%q due to error:%+v", status.String(), taskID, err)
		return nil
	}
	tasks.EmitTaskEventWithContextualLogs(ctx, targetID, taskID, taskType, wfName, status.String())
	if errReason != nil {
		return tasks.SetTaskErrorMessage(taskID, errReason.Error())
	}
	return nil
}

func buildTaskExecution(cc *api.Client, execID string) (*taskExecution, error) {
	kv := cc.KV()
	taskID, err := getExecutionKeyValue(execID, "taskID")
	if err != nil {
		return nil, err
	}
	targetID, err := tasks.GetTaskTarget(taskID)
	if err != nil {
		return nil, err
	}
	taskType, err := tasks.GetTaskType(taskID)
	if err != nil {
		return nil, err
	}

	// Retrieve workflow, Step information in case of workflow Step TaskExecution
	var step string
	if tasks.IsWorkflowTask(taskType) {
		step, err = getExecutionKeyValue(execID, "step")
		if err != nil {
			return nil, err
		}
	}

	creationDate := time.Now()
	creationDatePath := path.Join(consulutil.ExecutionsTaskPrefix, execID, "creationDate")
	kvp, _, err := kv.Get(creationDatePath, nil)
	if err != nil {
		return nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if kvp == nil || len(kvp.Value) == 0 {
		err = consulutil.StoreConsulKeyAsString(creationDatePath, creationDate.Format(time.RFC3339Nano))
		if err != nil {
			return nil, err
		}
	} else {
		creationDate, err = time.Parse(time.RFC3339Nano, string(kvp.Value))
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse creation date %q for execution %q", string(kvp.Value), execID)
		}
	}

	return newTaskExecution(execID, taskID, targetID, step, cc, creationDate, tasks.TaskType(taskType)), nil
}

// This may seams useless but as sometimes we create a fake task exec (like for actions) it force to provide all needed parameters
func newTaskExecution(id, taskID, targetID, step string, cc *api.Client, creationDate time.Time, taskType tasks.TaskType) *taskExecution {
	return &taskExecution{
		id:           id,
		taskID:       taskID,
		targetID:     targetID,
		cc:           cc,
		creationDate: creationDate,
		taskType:     tasks.TaskType(taskType),
		step:         step,
	}
}
