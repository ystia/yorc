package deployments

import (
	"context"
	"path"
	"reflect"
	"strings"
	"testing"

	"github.com/ystia/yorc/helper/consulutil"

	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/require"
	"github.com/ystia/yorc/tosca"
)

func testReadComplexVA(t *testing.T, kv *api.KV) {
	t.Parallel()
	deploymentID := strings.Replace(t.Name(), "/", "_", -1)
	err := StoreDeploymentDefinition(context.Background(), kv, deploymentID, "testdata/value_assignments.yaml")
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
			got, err := readComplexVA(kv, tt.args.vaType, deploymentID, tt.args.keyPath, tt.args.vaDatatype, tt.args.nestedKeys...)
			if (err != nil) != tt.wantErr {
				t.Errorf("readComplexVA() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("readComplexVA() = %v, want %v", got, tt.want)
			}
		})
	}
}
