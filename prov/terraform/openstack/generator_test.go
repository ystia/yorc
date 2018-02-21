package openstack

import (
	"testing"

	"encoding/json"

	"github.com/stretchr/testify/require"
	"github.com/ystia/yorc/prov/terraform/commons"
)

func Test_addOutput(t *testing.T) {
	type args struct {
		infrastructure *commons.Infrastructure
		outputName     string
		output         *commons.Output
	}
	tests := []struct {
		name       string
		args       args
		jsonResult string
	}{
		{"OneOutput", args{&commons.Infrastructure{}, "O1", &commons.Output{Value: "V1"}}, `{"output":{"O1":{"value":"V1"}}}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			commons.AddOutput(tt.args.infrastructure, tt.args.outputName, tt.args.output)
			res, err := json.Marshal(tt.args.infrastructure)
			require.Nil(t, err)
			require.Equal(t, tt.jsonResult, string(res))
		})
	}
}
