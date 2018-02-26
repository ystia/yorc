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
