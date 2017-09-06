package workflow

import (
	"path"
	"strconv"

	"time"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
	"novaforge.bull.com/starlings-janus/janus/tasks"
)

type task struct {
	ID           string
	TargetID     string
	status       tasks.TaskStatus
	TaskType     tasks.TaskType
	creationDate time.Time
	taskLock     *api.Lock
	kv           *api.KV
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
	if err != nil {
		return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	_, err = tasks.EmitTaskEvent(t.kv, t.TargetID, t.ID, t.TaskType, t.status.String())

	return err
}
