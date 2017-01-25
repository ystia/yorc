package workflow

import (
	"path"
	"strconv"

	"github.com/hashicorp/consul/api"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
	"novaforge.bull.com/starlings-janus/janus/tasks"
)

type task struct {
	ID       string
	TargetID string
	status   tasks.TaskStatus
	TaskType tasks.TaskType
	taskLock *api.Lock
	kv       *api.KV
}

func (t *task) releaseLock() {
	t.taskLock.Unlock()
	t.taskLock.Destroy()
}

func (t *task) Status() tasks.TaskStatus {
	return t.status
}

func (t *task) WithStatus(status tasks.TaskStatus) error {
	p := &api.KVPair{Key: path.Join(consulutil.TasksPrefix, t.ID, "status"), Value: []byte(strconv.Itoa(int(status)))}
	_, err := t.kv.Put(p, nil)
	t.status = status
	return err
}
