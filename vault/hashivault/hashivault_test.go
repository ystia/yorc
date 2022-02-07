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

package hashivault

import (
	"testing"

	"github.com/hashicorp/vault/api"
)

func Test_vaultSecret_String(t *testing.T) {
	type fields struct {
		Secret  *api.Secret
		options map[string]string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{"NoDataFormating", fields{Secret: &api.Secret{Data: map[string]interface{}{"foo": "bar"}}, options: map[string]string{}}, "map[foo:bar]"},
		{"WithDataFormatingOnV1", fields{Secret: &api.Secret{Data: map[string]interface{}{"foo": "bar"}}, options: map[string]string{"data": "foo"}}, "bar"},
		{"WithDataFormatingOnV2", fields{Secret: &api.Secret{Data: map[string]interface{}{
			"data":     map[string]interface{}{"foo": "bar"},
			"metadata": map[string]interface{}{"m": "d"},
		}}, options: map[string]string{"data": "foo"}}, "bar"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vs := &vaultSecret{
				Secret:  tt.fields.Secret,
				options: tt.fields.options,
			}
			if got := vs.String(); got != tt.want {
				t.Errorf("vaultSecret.String() = %v, want %v", got, tt.want)
			}
		})
	}
}
