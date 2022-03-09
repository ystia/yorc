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

package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"path"
	"reflect"
	"testing"

	"github.com/hashicorp/consul/sdk/testutil"
	"github.com/stretchr/testify/require"

	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/storage"
	"github.com/ystia/yorc/v4/storage/types"
	"github.com/ystia/yorc/v4/tosca"
)

func populateKV(t *testing.T, srv *testutil.TestServer) {
	ctx := context.Background()
	node3 := tosca.NodeTemplate{
		Type: "tosca.nodes.Compute",
	}

	err := storage.GetStore(types.StoreTypeDeployment).Set(ctx, consulutil.DeploymentKVPrefix+"/id1/topology/nodes/node3", node3)
	require.Nil(t, err)

	srv.PopulateKV(t, map[string][]byte{
		consulutil.TasksPrefix + "/t1/targetId": []byte("id1"),
		consulutil.TasksPrefix + "/t1/status":   []byte("0"),
		consulutil.TasksPrefix + "/t1/" +
			stepRegistrationInProgressKey: []byte("true"),
		consulutil.TasksPrefix + "/t1/type":             []byte("0"),
		consulutil.TasksPrefix + "/t1/data/inputs/i0":   []byte("0"),
		consulutil.TasksPrefix + "/t1/data/nodes/node1": []byte("0,1,2"),
		consulutil.TasksPrefix + "/t2/targetId":         []byte("id1"),
		consulutil.TasksPrefix + "/t2/status":           []byte("1"),
		consulutil.TasksPrefix + "/t2/type":             []byte("1"),
		consulutil.TasksPrefix + "/t3/targetId":         []byte("id2"),
		consulutil.TasksPrefix + "/t3/status":           []byte("2"),
		consulutil.TasksPrefix + "/t3/type":             []byte("2"),
		consulutil.TasksPrefix + "/t3/data/nodes/n1":    []byte("2"),
		consulutil.TasksPrefix + "/t3/data/nodes/n2":    []byte("2"),
		consulutil.TasksPrefix + "/t3/data/nodes/n3":    []byte("2"),
		consulutil.TasksPrefix + "/t4/targetId":         []byte("id1"),
		consulutil.TasksPrefix + "/t4/status":           []byte("3"),
		consulutil.TasksPrefix + "/t4/type":             []byte("3"),
		consulutil.TasksPrefix + "/t5/targetId":         []byte("id"),
		consulutil.TasksPrefix + "/t5/status":           []byte("4"),
		consulutil.TasksPrefix + "/t5/type":             []byte("4"),
		consulutil.TasksPrefix + "/tCustomWF/targetId":  []byte("id"),
		consulutil.TasksPrefix + "/tCustomWF/status":    []byte("0"),
		consulutil.TasksPrefix + "/tCustomWF/type":      []byte("6"),
		consulutil.TasksPrefix + "/t6/targetId":         []byte("id"),
		consulutil.TasksPrefix + "/t6/status":           []byte("5"),
		consulutil.TasksPrefix + "/t6/type":             []byte("5"),
		consulutil.TasksPrefix + "/t7/targetId":         []byte("id"),
		consulutil.TasksPrefix + "/t7/status":           []byte("5"),
		consulutil.TasksPrefix + "/t7/type":             []byte("6666"),
		consulutil.TasksPrefix + "/tNotInt/targetId":    []byte("targetNotInt"),
		consulutil.TasksPrefix + "/tNotInt/status":      []byte("not a status"),
		consulutil.TasksPrefix + "/tNotInt/type":        []byte("not a type"),

		consulutil.DeploymentKVPrefix + "/id1/topology/instances/node2/0/id": []byte("0"),
		consulutil.DeploymentKVPrefix + "/id1/topology/instances/node2/1/id": []byte("1"),

		consulutil.WorkflowsPrefix + "/t8/step1":  []byte("status1"),
		consulutil.WorkflowsPrefix + "/t8/step2":  []byte("status2"),
		consulutil.WorkflowsPrefix + "/t8/step3":  []byte("status3"),
		consulutil.WorkflowsPrefix + "/t10/step1": []byte("error"),
		consulutil.WorkflowsPrefix + "/t11/step1": []byte("status1"),

		consulutil.TasksPrefix + "/t12/status":    []byte("3"),
		consulutil.TasksPrefix + "/t12/type":      []byte("5"),
		consulutil.TasksPrefix + "/t12/targetId":  []byte("id"),
		consulutil.TasksPrefix + "/t13/resultSet": buildResultset(),
		consulutil.TasksPrefix + "/t13/targetId":  []byte("id"),
		consulutil.TasksPrefix + "/t13/type":      []byte("5"),
		consulutil.TasksPrefix + "/t13/status":    []byte("3"),
		consulutil.TasksPrefix + "/t14/status":    []byte("3"),
		consulutil.TasksPrefix + "/t14/type":      []byte("6"),
		consulutil.TasksPrefix + "/t14/targetId":  []byte("id"),

		consulutil.TasksPrefix + "/t15/targetId": []byte("xxx"),
		consulutil.TasksPrefix + "/t15/status":   []byte("2"),
		consulutil.TasksPrefix + "/t15/type":     []byte("2"),

		consulutil.TasksPrefix + "/t16/targetId": []byte("infra_usage:slurm"),
		consulutil.TasksPrefix + "/t16/status":   []byte("2"),
		consulutil.TasksPrefix + "/t16/type":     []byte("7"),
		consulutil.TasksPrefix + "/t17/targetId": []byte("infra_usage:bbb"),
		consulutil.TasksPrefix + "/t17/status":   []byte("2"),
		consulutil.TasksPrefix + "/t17/type":     []byte("7"),
		consulutil.TasksPrefix + "/t18/targetId": []byte("infra_usage:slurm"),
		consulutil.TasksPrefix + "/t18/status":   []byte("2"),
		consulutil.TasksPrefix + "/t18/type":     []byte("7"),
	})
}

func buildResultset() []byte {
	m := make(map[string]interface{})
	m["key1"] = "value1"
	m["key2"] = "value2"
	m["key3"] = "value3"

	res, err := json.Marshal(m)
	if err != nil {
		fmt.Printf("Failed to marshal map [%+v]: due to error:%+v", m, err)
	}
	return res
}

func testGetTasksIdsForTarget(t *testing.T) {
	type args struct {
		targetID string
	}
	tests := []struct {
		name    string
		args    args
		want    []string
		wantErr bool
	}{
		{"TestMultiTargets", args{"id1"}, []string{"t1", "t2", "t4"}, false},
		{"TestSingleTarget", args{"id2"}, []string{"t3"}, false},
		{"TestNoTarget", args{"idDoesntExist"}, []string{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetTasksIdsForTarget(tt.args.targetID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetTasksIdsForTarget() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetTasksIdsForTarget() = %v, want %v", got, tt.want)
			}
		})
	}
}

func testGetTaskStatus(t *testing.T) {
	type args struct {
		taskID string
	}
	tests := []struct {
		name    string
		args    args
		want    TaskStatus
		wantErr bool
	}{
		{"StatusINITIAL", args{"t1"}, TaskStatusINITIAL, false},
		{"StatusRUNNING", args{"t2"}, TaskStatusRUNNING, false},
		{"StatusDONE", args{"t3"}, TaskStatusDONE, false},
		{"StatusFAILED", args{"t4"}, TaskStatusFAILED, false},
		{"StatusCANCELED", args{"t5"}, TaskStatusCANCELED, false},
		{"StatusDoesntExist", args{"t6"}, TaskStatusFAILED, true},
		{"StatusNotInt", args{"tNotInt"}, TaskStatusFAILED, true},
		{"TaskDoesntExist", args{"TaskDoesntExist"}, TaskStatusFAILED, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetTaskStatus(tt.args.taskID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetTaskStatus() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetTaskStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}

func testGetTaskType(t *testing.T) {
	type args struct {
		taskID string
	}
	tests := []struct {
		name    string
		args    args
		want    TaskType
		wantErr bool
	}{
		{"TypeDeploy", args{"t1"}, TaskTypeDeploy, false},
		{"TypeUnDeploy", args{"t2"}, TaskTypeUnDeploy, false},
		{"TypeScaleOut", args{"t3"}, TaskTypeScaleOut, false},
		{"TypeScaleIn", args{"t4"}, TaskTypeScaleIn, false},
		{"TypePurge", args{"t5"}, TaskTypePurge, false},
		{"TypeCustomCommand", args{"t6"}, TaskTypeCustomCommand, false},
		{"TypeDoesntExist", args{"t7"}, TaskTypeDeploy, true},
		{"TypeNotInt", args{"tNotInt"}, TaskTypeDeploy, true},
		{"TaskDoesntExist", args{"TaskDoesntExist"}, TaskTypeDeploy, true},
		{"TypeCustomWorkflow", args{"tCustomWF"}, TaskTypeCustomWorkflow, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetTaskType(tt.args.taskID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetTaskType() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetTaskType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func testGetTaskTarget(t *testing.T) {
	type args struct {
		taskID string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{"GetTarget", args{"t1"}, "id1", false},
		{"TaskDoesntExist", args{"TaskDoesntExist"}, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetTaskTarget(tt.args.taskID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetTaskTarget() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetTaskTarget() = %v, want %v", got, tt.want)
			}
		})
	}
}

func testTaskExists(t *testing.T) {
	type args struct {
		taskID string
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{"TaskExist", args{"t1"}, true, false},
		{"TaskDoesntExist", args{"TaskDoesntExist"}, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := TaskExists(tt.args.taskID)
			if (err != nil) != tt.wantErr {
				t.Errorf("TaskExists() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("TaskExists() = %v, want %v", got, tt.want)
			}
		})
	}
}

func testCancelTask(t *testing.T) {
	type args struct {
		taskID string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"CancelTask", args{"t1"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := CancelTask(tt.args.taskID); (err != nil) != tt.wantErr {
				t.Errorf("CancelTask() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			kvp, _, err := consulutil.GetKV().Get(path.Join(consulutil.TasksPrefix, tt.args.taskID, ".canceledFlag"), nil)
			if err != nil {
				t.Errorf("Unexpected Consul communication error: %v", err)
				return
			}
			if kvp == nil {
				t.Error("canceledFlag missing")
				return
			}
			if string(kvp.Value) != "true" {
				t.Error("canceledFlag not set to \"true\"")
				return
			}
		})
	}
}

func testTargetHasLivingTasks(t *testing.T) {
	type args struct {
		targetID      string
		typesToIgnore []TaskType
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		want1   string
		want2   string
		wantErr bool
	}{
		{"TargetHasRunningTasks", args{"id1", []TaskType{}}, true, "t1", "INITIAL", false},
		{"TargetHasNoRunningDeployTasks", args{"id1",
			[]TaskType{TaskTypeDeploy, TaskTypeUnDeploy, TaskTypeScaleIn}}, false, "", "", false},
		{"TargetHasNoRunningTasks", args{"id2", []TaskType{}}, false, "", "", false},
		{"TargetDoesntExist", args{"TargetDoesntExist", []TaskType{}}, false, "", "", false},
		{"TargetNotInt", args{"targetNotInt", []TaskType{}}, false, "", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, got2, err := TargetHasLivingTasks(tt.args.targetID, tt.args.typesToIgnore)
			if (err != nil) != tt.wantErr {
				t.Errorf("TargetHasLivingTasks() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("TargetHasLivingTasks() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("TargetHasLivingTasks() got1 = %v, want %v", got1, tt.want1)
			}
			if got2 != tt.want2 {
				t.Errorf("TargetHasLivingTasks() got2 = %v, want %v", got2, tt.want2)
			}
		})
	}
}

func testGetTaskInput(t *testing.T) {
	type args struct {
		taskID    string
		inputName string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{"InputExist", args{"t1", "i0"}, "0", false},
		{"InputDoesnt", args{"t1", "i1"}, "", true},
		{"InputsDoesnt", args{"t2", "i1"}, "", true},
		{"TaskDoesntExist", args{"TargetDoesntExist", "i1"}, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetTaskInput(tt.args.taskID, tt.args.inputName)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetTaskInput() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetTaskInput() = %v, want %v", got, tt.want)
			}
		})
	}
}

func testGetInstances(t *testing.T) {
	type args struct {
		taskID       string
		deploymentID string
		nodeName     string
	}
	tests := []struct {
		name    string
		args    args
		want    []string
		wantErr bool
	}{
		{"TaskRelatedNodes", args{"t1", "id1", "node1"}, []string{"0", "1", "2"}, false},
		{"TaskRelatedNodes", args{"t1", "id1", "node2"}, []string{"0", "1"}, false},
		{"TaskRelatedNodes", args{"t2", "id1", "node2"}, []string{"0", "1"}, false},
		{"TaskDoesntExistDeploymentDoes", args{"TaskDoesntExist", "id1", "node2"}, []string{"0", "1"}, false},
		{"TaskDoesntExistDeploymentDoesInstanceDont", args{"TaskDoesntExist", "id1", "node3"}, []string{}, false},
		{"TaskDoesntExistDeploymentToo", args{"TaskDoesntExist", "idDoesntExist", "node2"}, []string{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetInstances(context.Background(), tt.args.taskID, tt.args.deploymentID, tt.args.nodeName)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetInstances() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetInstances() = %v, want %v", got, tt.want)
			}
		})
	}
}

func testGetTaskRelatedNodes(t *testing.T) {
	type args struct {
		taskID string
	}
	tests := []struct {
		name    string
		args    args
		want    []string
		wantErr bool
	}{
		{"TaskRelNodes", args{"t1"}, []string{"node1"}, false},
		{"NoTaskRelNodes", args{"t2"}, nil, false},
		{"NoTaskRelNodes", args{"t3"}, []string{"n1", "n2", "n3"}, false},
		{"TaskDoesntExist", args{"TaskDoesntExist"}, nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetTaskRelatedNodes(tt.args.taskID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetTaskRelatedNodes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetTaskRelatedNodes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func testIsTaskRelatedNode(t *testing.T) {
	type args struct {
		taskID   string
		nodeName string
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{"TaskRelNode", args{"t1", "node1"}, true, false},
		{"NotTaskRelNode", args{"t1", "node2"}, false, false},
		{"NotTaskRelNode2", args{"t2", "node1"}, false, false},
		{"NotTaskRelNode3", args{"t2", "node2"}, false, false},
		{"TaskDoesntExist", args{"TaskDoesntExist", "node2"}, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := IsTaskRelatedNode(tt.args.taskID, tt.args.nodeName)
			if (err != nil) != tt.wantErr {
				t.Errorf("IsTaskRelatedNode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("IsTaskRelatedNode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func testGetTaskRelatedWFSteps(t *testing.T) {
	type args struct {
		taskID string
	}
	tests := []struct {
		name    string
		args    args
		want    []TaskStep
		wantErr bool
	}{
		{"TaskWith3Steps", args{"t8"}, []TaskStep{{Name: "step1", Status: "status1"}, {Name: "step2", Status: "status2"}, {Name: "step3", Status: "status3"}}, false},
		{"TaskWithoutStep", args{"t9"}, []TaskStep{}, false},
		{"TaskDoesntExist", args{"fake"}, []TaskStep{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetTaskRelatedSteps(tt.args.taskID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetTaskRelatedSteps() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			require.Len(t, got, len(tt.want), "GetTaskRelatedSteps()  %d task steps expected instead of %d", len(tt.want), len(got))
			for _, wanted := range tt.want {
				require.Contains(t, got, wanted, "GetTaskRelatedSteps() %v expected", wanted)
			}
		})
	}
}

func testUpdateTaskStepStatus(t *testing.T) {
	type args struct {
		taskID string
		step   *TaskStep
	}
	tests := []struct {
		name    string
		args    args
		want    *TaskStep
		wantErr bool
	}{
		{"TaskStep", args{"t10", &TaskStep{Name: "step1", Status: "done"}}, &TaskStep{Name: "step1", Status: "DONE"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			UpdateTaskStepStatus(tt.args.taskID, tt.want)
			_, found, err := TaskStepExists(tt.args.taskID, tt.args.step.Name)
			if (err != nil) != tt.wantErr {
				t.Errorf("TaskStepExists() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(found, tt.want) {
				t.Errorf("TaskStepExists() = %v, want %v", found, tt.want)
			}
		})
	}
}

func TestCheckTaskStepStatusChange(t *testing.T) {
	type args struct {
		before string
		after  string
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{"ChangeOK", args{"error", "done"}, true, false},
		{"NotAllowed", args{"initial", "done"}, false, false},
		{"NotAllowed", args{"initial", "running"}, false, false},
		{"Error", args{"fake", "fake"}, false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, err := CheckTaskStepStatusChange(tt.args.before, tt.args.after)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckTaskStepStatusChange() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if ok != tt.want {
				t.Errorf("CheckTaskStepStatusChange() = %v, want %v", ok, tt.want)
			}
		})
	}
}

func testTaskStepExists(t *testing.T) {
	type args struct {
		taskID string
		stepID string
	}
	type ret struct {
		found bool
		step  *TaskStep
	}
	tests := []struct {
		name    string
		args    args
		want    ret
		wantErr bool
	}{
		{"TaskStep", args{"t11", "step1"}, ret{true, &TaskStep{Name: "step1", Status: "status1"}}, false},
		{"TaskStep", args{"fake", "fakeAgain"}, ret{false, nil}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exist, stepFound, err := TaskStepExists(tt.args.taskID, tt.args.stepID)
			if (err != nil) != tt.wantErr {
				t.Errorf("testTaskStepExists() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			result := ret{exist, stepFound}
			if !reflect.DeepEqual(result, tt.want) {
				t.Errorf("testTaskStepExists() = %v, want %v", result, tt.want)
			}
		})
	}
}

func testGetTaskResultSet(t *testing.T) {
	type args struct {
		taskID string
	}

	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{"taskWithResultSet", args{"t13"}, "{\"key1\":\"value1\",\"key2\":\"value2\",\"key3\":\"value3\"}", false},
		{"taskWithoutResultSet", args{"t14"}, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetTaskResultSet(tt.args.taskID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetTaskStatus() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetTaskStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}

func testDeleteTask(t *testing.T) {
	type args struct {
		taskID string
	}

	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{"taskToDelete", args{"t15"}, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := DeleteTask(tt.args.taskID)
			if (err != nil) != tt.wantErr {
				t.Errorf("DeleteTask() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			got, err := TaskExists(tt.args.taskID)
			if (err != nil) != tt.wantErr {
				t.Errorf("DeleteTask() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("DeleteTask() = %v, want %v", got, tt.want)
			}
		})
	}
}

func testGetQueryTaskIDs(t *testing.T) {
	type args struct {
		taskID string
	}

	tests := []struct {
		name    string
		args    args
		want    []string
		wantErr bool
	}{
		{"taskWithTypeQueryAndInfraUsage", args{"t13"}, []string{"t16", "t18"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetQueryTaskIDs(TaskTypeQuery, "infra_usage", "slurm")
			if (err != nil) != tt.wantErr {
				t.Errorf("GetQueryTaskIDs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetQueryTaskIDs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func testIsStepRegistrationInProgress(t *testing.T) {
	type args struct {
		taskID string
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{"RegistrationInProgress", args{"t1"}, true, false},
		{"NoRegistrationInProgress", args{"t2"}, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := IsStepRegistrationInProgress(tt.args.taskID)
			if (err != nil) != tt.wantErr {
				t.Errorf("%s error = %v, wantErr %v", tt.name, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("%s = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func testCheckAndSetTaskErrorMessage(t *testing.T) {
	type setupFn func(t *testing.T, taskID string)
	type args struct {
		errorMessage      string
		overwriteExisting bool
	}
	tests := []struct {
		name                 string
		args                 args
		setupFn              setupFn
		wantErr              bool
		expectedErrorMessage string
	}{
		{"TaskExistsErrorMessageDoesNotExistNoOverwrite", args{"MyMessage", false}, func(t *testing.T, taskID string) {
			consulutil.StoreConsulKeyAsString(path.Join(consulutil.TasksPrefix, taskID, "targetId"), "id")
			consulutil.StoreConsulKeyAsString(path.Join(consulutil.TasksPrefix, taskID, "status"), "1")
			consulutil.StoreConsulKeyAsString(path.Join(consulutil.TasksPrefix, taskID, "type"), "1")
		}, false, "MyMessage"},
		{"TaskExistsErrorMessageDoesNotExistOverwrite", args{"MyMessage", true}, func(t *testing.T, taskID string) {
			consulutil.StoreConsulKeyAsString(path.Join(consulutil.TasksPrefix, taskID, "targetId"), "id")
			consulutil.StoreConsulKeyAsString(path.Join(consulutil.TasksPrefix, taskID, "status"), "1")
			consulutil.StoreConsulKeyAsString(path.Join(consulutil.TasksPrefix, taskID, "type"), "1")
		}, false, "MyMessage"},
		{"TaskExistsErrorMessageExistNoOverwrite", args{"MyMessage", false}, func(t *testing.T, taskID string) {
			consulutil.StoreConsulKeyAsString(path.Join(consulutil.TasksPrefix, taskID, "targetId"), "id")
			consulutil.StoreConsulKeyAsString(path.Join(consulutil.TasksPrefix, taskID, "status"), "1")
			consulutil.StoreConsulKeyAsString(path.Join(consulutil.TasksPrefix, taskID, "type"), "1")
			consulutil.StoreConsulKeyAsString(path.Join(consulutil.TasksPrefix, taskID, "errorMessage"), "PreviousErrorMessage")
		}, false, "PreviousErrorMessage"},
		{"TaskExistsErrorMessageExistOverwrite", args{"MyMessage", true}, func(t *testing.T, taskID string) {
			consulutil.StoreConsulKeyAsString(path.Join(consulutil.TasksPrefix, taskID, "targetId"), "id")
			consulutil.StoreConsulKeyAsString(path.Join(consulutil.TasksPrefix, taskID, "status"), "1")
			consulutil.StoreConsulKeyAsString(path.Join(consulutil.TasksPrefix, taskID, "type"), "1")
			consulutil.StoreConsulKeyAsString(path.Join(consulutil.TasksPrefix, taskID, "errorMessage"), "PreviousErrorMessage")
		}, false, "MyMessage"},
		{"TaskDoesNotExistNoOverwrite", args{"MyMessage", false}, nil, false, "MyMessage"},
		{"TaskDoesNotExistOverwrite", args{"MyMessage", true}, nil, false, "MyMessage"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			taskID := url.PathEscape(t.Name())
			defer consulutil.Delete(path.Join(consulutil.TasksPrefix, taskID), true)
			if tt.setupFn != nil {
				tt.setupFn(t, taskID)
			}
			if err := CheckAndSetTaskErrorMessage(taskID, tt.args.errorMessage, tt.args.overwriteExisting); (err != nil) != tt.wantErr {
				t.Errorf("CheckAndSetTaskErrorMessage() error = %v, wantErr %v", err, tt.wantErr)
			}
			actualErrorMessage, err := GetTaskErrorMessage(taskID)
			require.NoError(t, err)
			if actualErrorMessage != tt.expectedErrorMessage {
				t.Errorf("CheckAndSetTaskErrorMessage() actualErrorMessage = %v, expectedErrorMessage %v", actualErrorMessage, tt.expectedErrorMessage)
			}

		})
	}
}
