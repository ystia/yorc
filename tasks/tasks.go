package tasks

import (
	"fmt"
	"path"
	"strconv"

	"strings"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"novaforge.bull.com/starlings-janus/janus/deployments"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
)

// GetTasksIdsForTarget returns IDs of tasks related to a given targetID
func GetTasksIdsForTarget(kv *api.KV, targetID string) ([]string, error) {
	tasksKeys, _, err := kv.Keys(consulutil.TasksPrefix+"/", "/", nil)
	if err != nil {
		return nil, err
	}
	tasks := make([]string, 0)
	for _, taskKey := range tasksKeys {
		kvp, _, err := kv.Get(path.Join(taskKey, "targetId"), nil)
		if err != nil {
			return nil, err
		}
		if kvp != nil && len(kvp.Value) > 0 && string(kvp.Value) == targetID {
			tasks = append(tasks, path.Base(taskKey))
		}
	}
	return tasks, nil
}

// GetTaskStatus retrieves the TaskStatus of a task
func GetTaskStatus(kv *api.KV, taskID string) (TaskStatus, error) {
	kvp, _, err := kv.Get(path.Join(consulutil.TasksPrefix, taskID, "status"), nil)
	if err != nil {
		return FAILED, err
	}
	if kvp == nil || len(kvp.Value) == 0 {
		return FAILED, fmt.Errorf("Missing status for task with id %q", taskID)
	}
	statusInt, err := strconv.Atoi(string(kvp.Value))
	if err != nil {
		return FAILED, err
	}
	return TaskStatus(statusInt), nil
}

// GetTaskType retrieves the TaskType of a task
func GetTaskType(kv *api.KV, taskID string) (TaskType, error) {
	kvp, _, err := kv.Get(path.Join(consulutil.TasksPrefix, taskID, "type"), nil)
	if err != nil {
		return Deploy, err
	}
	if kvp == nil || len(kvp.Value) == 0 {
		return Deploy, fmt.Errorf("Missing status for type with id %q", taskID)
	}
	typeInt, err := strconv.Atoi(string(kvp.Value))
	if err != nil {
		return Deploy, err
	}
	return TaskType(typeInt), nil
}

// GetTaskTarget retrieves the targetID of a task
func GetTaskTarget(kv *api.KV, taskID string) (string, error) {
	kvp, _, err := kv.Get(path.Join(consulutil.TasksPrefix, taskID, "targetId"), nil)
	if err != nil {
		return "", nil
	}
	if kvp == nil || len(kvp.Value) == 0 {
		return "", fmt.Errorf("Missing targetId for task with id %q", taskID)
	}
	return string(kvp.Value), nil
}

// TaskExists checks if a task with the given taskID exists
func TaskExists(kv *api.KV, taskID string) (bool, error) {
	kvp, _, err := kv.Get(path.Join(consulutil.TasksPrefix, taskID, "targetId"), nil)
	if err != nil {
		return false, nil
	}
	if kvp == nil || len(kvp.Value) == 0 {
		return false, nil
	}
	return true, nil
}

// CancelTask marks a task as Canceled
func CancelTask(kv *api.KV, taskID string) error {
	kvp := &api.KVPair{Key: path.Join(consulutil.TasksPrefix, taskID, ".canceledFlag"), Value: []byte("true")}
	_, err := kv.Put(kvp, nil)
	return err
}

// TargetHasLivingTasks checks if a targetID has associated tasks and returns their id and status
func TargetHasLivingTasks(kv *api.KV, targetID string) (bool, string, string, error) {
	tasksKeys, _, err := kv.Keys(consulutil.TasksPrefix+"/", "/", nil)
	if err != nil {
		return false, "", "", err
	}
	for _, taskKey := range tasksKeys {
		kvp, _, err := kv.Get(path.Join(taskKey, "targetId"), nil)
		if err != nil {
			return false, "", "", err
		}
		if kvp != nil && len(kvp.Value) > 0 && string(kvp.Value) == targetID {
			kvp, _, err := kv.Get(path.Join(taskKey, "status"), nil)
			taskID := path.Base(taskKey)
			if err != nil {
				return false, "", "", err
			}
			if kvp == nil || len(kvp.Value) == 0 {
				return false, "", "", fmt.Errorf("Missing status for task with id %q", taskID)
			}
			statusInt, err := strconv.Atoi(string(kvp.Value))
			if err != nil {
				return false, "", "", err
			}
			switch TaskStatus(statusInt) {
			case INITIAL, RUNNING:
				return true, taskID, TaskStatus(statusInt).String(), nil
			}
		}
	}
	return false, "", "", nil
}

// GetTaskInput retrieves inputs for tasks
func GetTaskInput(kv *api.KV, taskID, inputName string) (string, error) {
	kvP, _, err := kv.Get(path.Join(consulutil.TasksPrefix, taskID, "inputs", inputName), nil)
	if err != nil {
		return "", errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if kvP == nil {
		return "", errors.Errorf("Input %q not found for task %q", inputName, taskID)
	}
	return string(kvP.Value), nil
}

// GetInstances retrieve instances in the context of this task.
//
// Basically it checks if a list of instances is defined for this task for example in case of scaling.
// If not found it will returns the result of deployments.GetNodeInstancesIds(kv, deploymentID, nodeName).
func GetInstances(kv *api.KV, taskID, deploymentID, nodeName string) ([]string, error) {
	kvp, _, err := kv.Get(path.Join(consulutil.TasksPrefix, taskID, "nodes", nodeName), nil)
	if err != nil {
		return nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if kvp == nil || len(kvp.Value) == 0 {
		return deployments.GetNodeInstancesIds(kv, deploymentID, nodeName)
	}
	return strings.Split(string(kvp.Value), ","), nil
}

// GetTaskRelatedNodes returns the list of nodes that are specifically targeted by this task
//
// Currently it only appens for scaling tasks
func GetTaskRelatedNodes(kv *api.KV, taskID string) ([]string, error) {
	nodes, _, err := kv.Keys(path.Join(consulutil.TasksPrefix, taskID, "nodes")+"/", "/", nil)
	if err != nil {
		return nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	for i := range nodes {
		nodes[i] = path.Dir(nodes[i])
	}
	return nodes, nil
}

// IsTaskRelatedNode checks if the given nodeName is declared as a task related node
func IsTaskRelatedNode(kv *api.KV, taskID, nodeName string) (bool, error) {
	kvp, _, err := kv.Get(path.Join(consulutil.TasksPrefix, taskID, "nodes", nodeName), nil)
	if err != nil {
		return false, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	return kvp != nil, nil
}
