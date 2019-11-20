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

package deployments

import (
	"context"
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/tosca"
)

func testReadComplexVA(t *testing.T) {
	// t.Parallel()
	deploymentID := strings.Replace(t.Name(), "/", "_", -1)
	err := StoreDeploymentDefinition(context.Background(), deploymentID, "testdata/value_assignments.yaml")
	require.Nil(t, err)

	type args struct {
		vaType     tosca.ValueAssignmentType
		keyPath    string
		vaDatatype string
		nestedKeys []string
	}
	tests := []struct {
		name    string
		args    args
		want    interface{}
		wantErr bool
	}{
		{"ReadComplexVASimpleCase", args{tosca.ValueAssignmentMap, path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes/VANode1/properties/map"), "map:string", nil}, map[string]interface{}{"one": "1", "two": "2"}, false},
		{"ReadComplexVAAllSet", args{tosca.ValueAssignmentMap, path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes/VANode1/properties/complex"), "yorc.tests.datatypes.ComplexType", nil}, map[string]interface{}{"literal": "11", "literalDefault": "VANode1LitDef"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			//got, err := readComplexVA(context.Background(), tt.args.vaType, deploymentID, tt.args.keyPath, tt.args.vaDatatype, tt.args.nestedKeys...)
			//if (err != nil) != tt.wantErr {
			//	t.Errorf("readComplexVA() error = %v, wantErr %v", err, tt.wantErr)
			//	return
			//}
			//if !reflect.DeepEqual(got, tt.want) {
			//	t.Errorf("readComplexVA() = %v, want %v", got, tt.want)
			//}
		})
	}
}
