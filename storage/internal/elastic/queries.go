// Copyright 2019 Bull S.A.S. Atos Technologies - Bull, Rue Jean Jaures, B.P.68, 78340, Les Clayes-sous-Bois, France.
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

package elastic

import (
	"bytes"
	"strconv"
	"text/template"
)

// Index creation request
const initStorageTemplateText = `
{
     "settings": {
        {{ if ne .InitialReplicas -1}}"number_of_replicas": {{ .InitialReplicas}},{{end}}            
        {{ if ne .InitialShards -1 }}"number_of_shards": {{ .InitialShards}},{{end}}
        "refresh_interval": "1s"
     },
     "mappings": {
         "_doc": {
             "_all": {"enabled": false},
             "dynamic": "false",
             "properties": {
                 "deploymentId": { "type": "keyword", "index": true },
                 "iid": { "type": "long", "index": true },
                 "iidStr": { "type": "keyword","index": false }
             }
         }
     }
}`

// Get last Modified index
const lastModifiedIndexTemplateText = `
{
    "aggs" : {
        "max_iid" : {
            "filter" : {
{{if .}}
                "bool": {
                    "must": [
                        { "term": { "deploymentId": "{{ . }}" } }
                     ]
                }
{{else}}
                "match_all": {}
{{end}}
            },
            "aggs" : {
                "last_index" : { "max" : { "field" : "iid" } }
            }
        }
    }
}`

// Range Query
const rangeQueryTemplateText = `{ "range":{ "iid":{ "gt": "{{ conv .WaitIndex }}"{{if gt .MaxIndex 0}},"lte": "{{ conv .MaxIndex }}"{{end}}}}}`

const listQueryTemplateText = `
{
  "query":{{if .DeploymentID}}{
    "bool":{
        "must": [
          {
            "term":{
               "deploymentId": "{{ .DeploymentID }}"
            }
          },
          {{template "rangeQuery" .}}
        ]
    }
  }{{else}}{{template "rangeQuery" .}}{{end}}
}
`

var templates *template.Template

func init() {
	funcMap := template.FuncMap{"conv": func(value uint64) string { return strconv.FormatUint(value, 10) }}

	templates = template.Must(template.New("initStorage").Parse(initStorageTemplateText))
	templates = template.Must(templates.New("lastModifiedIndex").Parse(lastModifiedIndexTemplateText))

	templates = template.Must(templates.New("rangeQuery").Funcs(funcMap).Parse(rangeQueryTemplateText))
	templates = template.Must(templates.New("listQuery").Parse(listQueryTemplateText))
}

// Return the query that is used to create indexes for event and log storage.
// We only index the needed fields to optimize ES indexing performance (no dynamic mapping).
func buildInitStorageIndexQuery(elasticStoreConfig elasticStoreConf) string {
	var buffer bytes.Buffer
	templates.ExecuteTemplate(&buffer, "initStorage", elasticStoreConfig)
	return buffer.String()
}

// This ES aggregation query is built using clusterId and eventually deploymentId.
func buildLastModifiedIndexQuery(deploymentID string) (query string) {
	var buffer bytes.Buffer
	templates.ExecuteTemplate(&buffer, "lastModifiedIndex", deploymentID)
	return buffer.String()
}

// This ES range query is built using 'waitIndex' and eventually 'maxIndex' and filtered using 'clusterId' and eventually 'deploymentId'.
func getListQuery(deploymentID string, waitIndex uint64, maxIndex uint64) (query string) {
	var buffer bytes.Buffer

	data := struct {
		WaitIndex    uint64
		MaxIndex     uint64
		DeploymentID string
	}{
		WaitIndex:    waitIndex,
		MaxIndex:     maxIndex,
		DeploymentID: deploymentID,
	}

	templates.ExecuteTemplate(&buffer, "listQuery", data)
	return buffer.String()
}
