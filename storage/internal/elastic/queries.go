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

import "strconv"

// Return the query that is used to create indexes for event and log storage.
// We only index the needed fields to optimize ES indexing performance (no dynamic mapping).
func buildInitStorageIndexQuery() (query string) {
	query = `
{
     "settings": {
         "refresh_interval": "1s"
     },
     "mappings": {
         "_doc": {
             "_all": {"enabled": false},
             "dynamic": "false",
             "properties": {
                 "clusterId": {
                     "type": "keyword",
                     "index": true
                 },
                 "deploymentId": {
                     "type": "keyword",
                     "index": true
                 },
                 "iid": {
                     "type": "long",
                     "index": true
                 },
                 "iidStr": {
                     "type": "keyword",
                     "index": false
                 }
             }
         }
     }
}`
	return
}

// This ES aggregation query is built using clusterId and eventually deploymentId.
func buildLastModifiedIndexQuery(clusterID string, deploymentID string) (query string) {
	if len(deploymentID) == 0 {
		query = `
{
    "aggs" : {
        "max_iid" : {
            "filter" : {
				"term": { "clusterId": "` + clusterID + `" }
            },
            "aggs" : {
                "last_index" : { "max" : { "field" : "iid" } }
            }
        }
    }
}`
	} else {
		query = `
{
    "aggs" : {
        "max_iid" : {
            "filter" : {
                "bool": {
                    "must": [
                        { "term": { "deploymentId": "` + deploymentID + `" } },
                        { "term": { "clusterId": "` + clusterID + `" } }
                     ]
                }
            },
            "aggs" : {
                "last_index" : { "max" : { "field" : "iid" } }
            }
        }
    }
}`
	}
	return
}

func getRangeQuery(waitIndex uint64, maxIndex uint64) (rangeQuery string) {
	if maxIndex > 0 {
		rangeQuery = `
            {
               "range":{
                  "iid":{
                     "gt": "` + strconv.FormatUint(waitIndex, 10) + `",
					 "lte": "` + strconv.FormatUint(maxIndex, 10) + `"
                  }
               }
            }`
	} else {
		rangeQuery = `
            {
               "range":{
                  "iid":{
                     "gt": "` + strconv.FormatUint(waitIndex, 10) + `"
                  }
               }
            }`
	}
	return
}

// This ES range query is built using 'waitIndex' and eventually 'maxIndex' and filtered using 'clusterId' and eventually 'deploymentId'.
func getListQuery(clusterID string, deploymentID string, waitIndex uint64, maxIndex uint64) (query string) {
	rangeQuery := getRangeQuery(waitIndex, maxIndex)
	if len(deploymentID) == 0 {
		query = `
{
   "query":{
      "bool":{
         "must":[
            {
               "term":{
                  "clusterId":"` + clusterID + `"
               }
            },` + rangeQuery + `
         ]
      }
   }
}`
	} else {
		query = `
{
   "query":{
      "bool":{
         "must":[
            {
               "term":{
                  "clusterId":"` + clusterID + `"
               }
            },
            {
               "term":{
                  "deploymentId":"` + deploymentID + `"
               }
            },` + rangeQuery + `
         ]
      }
   }
}`
	}
	return
}
