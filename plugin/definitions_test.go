package plugin

import (
	"reflect"
	"testing"

	plugin "github.com/hashicorp/go-plugin"
	"github.com/stretchr/testify/require"
)

func createClientServer(t *testing.T, opts *ServeOpts) (*plugin.RPCClient, *plugin.RPCServer) {
	return plugin.TestPluginRPCConn(t, getPlugins(opts))
}

func TestDefinitionsClient_GetDefinitions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		want    map[string][]byte
		wantErr bool
	}{
		{"TestOneDefinition", map[string][]byte{"TestOne": []byte{0}}, false},
		{"TestThreeDefinitions", map[string][]byte{"TestOne": []byte{0}, "TestTwo": []byte{0, 1}, "TestThree": []byte{0, 1, 2}}, false},
		{"TestNoDefinitions", map[string][]byte{}, false},
		{"TestNilDefinitions", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c, _ := createClientServer(t, &ServeOpts{Definitions: tt.want})
			defer c.Close()
			raw, err := c.Dispense(DefinitionsPluginName)
			require.Nil(t, err)
			def := raw.(Definitions)
			got, err := def.GetDefinitions()
			if (err != nil) != tt.wantErr {
				t.Errorf("DefinitionsClient.GetDefinitions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DefinitionsClient.GetDefinitions() = %v, want %v", got, tt.want)
			}
		})
	}
}
