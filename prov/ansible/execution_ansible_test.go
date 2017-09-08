package ansible

import (
	"os"
	"path/filepath"
	"testing"
	"text/template"

	"github.com/stretchr/testify/require"
	"novaforge.bull.com/starlings-janus/janus/prov"
)

func TestAnsibleTemplate(t *testing.T) {
	t.Parallel()
	ec := &executionCommon{
		NodeName:            "Welcome",
		operation:           prov.Operation{Name: "tosca.interfaces.node.lifecycle.standard.start"},
		Artifacts:           map[string]string{"scripts": "my_scripts", "s2": "somepath/sdq"},
		OverlayPath:         "/some/local/path",
		VarInputsNames:      []string{"INSTANCE", "PORT"},
		OperationRemotePath: ".janus/path/on/remote",
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
	tmpl, err := tmpl.Parse(ansiblePlaybook)
	require.Nil(t, err)
	err = tmpl.Execute(os.Stdout, e)
	t.Log(err)
	require.Nil(t, err)

}
