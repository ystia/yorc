package ansible

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
	"text/template"
)

func TestTemplates(t *testing.T) {
	e := &execution{Inputs: []string{"A: 1", "B: 2", "C: 3"}, NodeName: "Welcome", Operation: "tosca.interfaces.node.lifecycle.Standard.start", Artifacts: map[string]string{"scripts": "my_scripts"}, OverlayPath: "/some/local/path"}
	tmpl := template.New("execTest")
	tmpl = tmpl.Delims("[[[", "]]]")
	tmpl, err := tmpl.Parse(ansible_playbook)
	assert.Nil(t, err)
	err = tmpl.Execute(os.Stdout, e)
	t.Log(err)
	assert.Nil(t, err)
}
