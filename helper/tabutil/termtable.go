package tabutil

import (
	"fmt"

	"github.com/stevedomin/termtable"
)

type Table interface {
	AddRow(items ...interface{})
	AddHeaders(headers ...string)
	Render() string
}

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
