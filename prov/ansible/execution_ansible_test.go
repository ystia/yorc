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

package ansible

import (
	"os"
	"path/filepath"
	"testing"
	"text/template"

	"github.com/stretchr/testify/require"
	"github.com/ystia/yorc/prov"
)

func TestAnsibleTemplate(t *testing.T) {
	t.Parallel()
	ec := &executionCommon{
		NodeName:            "Welcome",
		operation:           prov.Operation{Name: "tosca.interfaces.node.lifecycle.standard.start"},
		Artifacts:           map[string]string{"scripts": "my_scripts", "s2": "somepath/sdq"},
		OverlayPath:         "/some/local/path",
		VarInputsNames:      []string{"INSTANCE", "PORT"},
		OperationRemotePath: ".yorc/path/on/remote",
	}

	e := &executionAnsible{
		PlaybookPath:    "/some/other/path.yml",
		executionCommon: ec,
	}

	funcMap := template.FuncMap{
		// The name "path" is what the function will be called in the template text.
		"path": filepath.Dir,
	}
	tmpl := template.New("execTest").Funcs(funcMap)
	tmpl = tmpl.Delims("[[[", "]]]")
	tmpl, err := tmpl.Parse(uploadArtifactsPlaybook + ansiblePlaybook)
	require.Nil(t, err)
	err = tmpl.Execute(os.Stdout, e)
	t.Log(err)
	require.Nil(t, err)

}
