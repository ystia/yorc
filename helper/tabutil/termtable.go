package tabutil

import (
	"fmt"

	"github.com/stevedomin/termtable"
)

// A Table allows to render console text in a table presentation
type Table interface {
	// AddRow adds a new line to the table
	AddRow(items ...interface{})
	// AddHeaders adds headers to the table
	AddHeaders(headers ...string)
	// Render renders the table and returns its string representation
	Render() string
}

// NewTable creates a Table
func NewTable() Table {
	return &termTable{tt: termtable.NewTable(nil, &termtable.TableOptions{
		Padding:      1,
		UseSeparator: true,
	})}
}

type termTable struct {
	tt *termtable.Table
}

func (t *termTable) AddHeaders(headers ...string) {
	t.tt.SetHeader(headers)
}

func (t *termTable) AddRow(items ...interface{}) {
	its := make([]string, len(items))
	for idx, item := range items {
		its[idx] = fmt.Sprint(item)
	}
	t.tt.AddRow(its)
}

func (t *termTable) Render() string {
	return t.tt.Render()
}
