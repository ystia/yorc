package ansible

import (
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"testing"
	"text/template"
)

func TestGroupedVolumeParallel(t *testing.T)  {
	t.Run("group", func(t *testing.T) {
		t.Run("templatesTest", templatesTest)
	})
}

func templatesTest(t *testing.T) {
	t.Parallel()
	e := &execution{Inputs: []string{"A: 1", "B: 2", "C: 3"}, NodeName: "Welcome", Operation: "tosca.interfaces.node.lifecycle.Standard.start", Artifacts: map[string]string{"scripts": "my_scripts"}, OverlayPath: "/some/local/path"}

	funcMap := template.FuncMap{
		// The name "path" is what the function will be called in the template text.
		"path": filepath.Dir,
	}

	tmpl := template.New("execTest")
	tmpl = tmpl.Delims("[[[", "]]]")
	tmpl = tmpl.Funcs(funcMap)
	tmpl, err := tmpl.Parse(ansible_playbook)
	assert.Nil(t, err)
	err = tmpl.Execute(os.Stdout, e)
	t.Log(err)
	assert.Nil(t, err)
}
