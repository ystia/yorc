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
	"context"
	"encoding/json"
	elasticsearch6 "github.com/elastic/go-elasticsearch/v6"
	"github.com/elastic/go-elasticsearch/v6/esapi"
	"github.com/pkg/errors"
	"github.com/ystia/yorc/v4/log"
	"github.com/ystia/yorc/v4/storage/store"
	"strings"
)

var pfalse = false

// structs for lastIndexRequest response decoding.
type lastIndexResponse struct {
	hits         hits                  `json:"hits"`
	aggregations logOrEventAggregation `json:"aggregations"`
}
type hits struct {
	total int `json:"total"`
}
type logOrEventAggregation struct {
	logsOrEvents lastIndexAggregation `json:"logs_or_events"`
}
type lastIndexAggregation struct {
	lastIndex stringValue `json:"last_index"`
}
type stringValue struct {
	value string `json:"value"`
}

// Init ES index for logs or events storage: create it if not found.
func initStorageIndex(c *elasticsearch6.Client, indexName string) {
	log.Printf("Checking if index <%s> already exists", indexName)

	// check if the sequences index exists
	req := esapi.IndicesExistsRequest{
		Index:           []string{indexName},
		ExpandWildcards: "none",
		AllowNoIndices:  &pfalse,
	}
	res, err := req.Do(context.Background(), c)
	debugESResponse("IndicesExistsRequest:"+indexName, res, err)
	defer res.Body.Close()

	if res.StatusCode == 200 {
		log.Printf("Indice %s was found, nothing to do !", indexName)
	}

	if res.StatusCode == 404 {
		log.Printf("Indice %s was not found, let's create it !", indexName)

		requestBodyData := buildInitStorageIndexQuery()

		// indice doest not exist, let's create it
		req := esapi.IndicesCreateRequest{
			Index: indexName,
			Body:  strings.NewReader(requestBodyData),
		}
		res, err := req.Do(context.Background(), c)
		debugESResponse("IndicesCreateRequest:"+indexName, res, err)
		defer res.Body.Close()
		if res.IsError() {
			var rsp map[string]interface{}
			json.NewDecoder(res.Body).Decode(&rsp)
			log.Printf("Response for IndicesCreateRequest (%s) : %+v", indexName, rsp)
		}
	}
}

// Perform a refresh query on ES cluster for this particular index.
func refreshIndex(c *elasticsearch6.Client, indexName string) {
	req := esapi.IndicesRefreshRequest{
		Index:           []string{indexName},
		ExpandWildcards: "none",
		AllowNoIndices:  &pfalse,
	}
	res, err := req.Do(context.Background(), c)
	defer res.Body.Close()
	debugESResponse("IndicesRefreshRequest:"+indexName, res, err)
}

// Just to display index settings at startup.
func debugIndexSetting(c *elasticsearch6.Client, indexName string) {
	log.Debugf("Get settings for index <%s>", indexName)
	req := esapi.IndicesGetSettingsRequest{
		Index:  []string{indexName},
		Pretty: true,
	}
	res, err := req.Do(context.Background(), c)
	debugESResponse("IndicesGetSettingsRequest:"+indexName, res, err)
	defer res.Body.Close()
}

// Debug the ES response.
func debugESResponse(msg string, res *esapi.Response, err error) {
	if err != nil {
		log.Debugf("[%s] Error while requesting ES : %+v", msg, err)
	} else if res.IsError() {
		var rsp map[string]interface{}
		json.NewDecoder(res.Body).Decode(&rsp)
		log.Debugf("[%s] Response Error while requesting ES (%d): %+v", msg, res.StatusCode, rsp)
	} else {
		var rsp map[string]interface{}
		json.NewDecoder(res.Body).Decode(&rsp)
		log.Debugf("[%s] Success ES response (%d): %+v", msg, res.StatusCode, rsp)
	}
}

// Query ES for events or logs specifying the expected results 'size' and the sort 'order'.
func doQueryEs(c *elasticsearch6.Client, index string, query string, waitIndex uint64, size int, order string) (int, []store.KeyValueOut, uint64, error) {
	log.Debugf("Search ES %s using query: %s", index, query)

	values := make([]store.KeyValueOut, 0)

	res, err := c.Search(
		c.Search.WithContext(context.Background()),
		c.Search.WithIndex(index),
		c.Search.WithSize(size),
		c.Search.WithBody(strings.NewReader(query)),
		// important sort on iid
		c.Search.WithSort("iid:"+order),
	)
	if err != nil {
		log.Printf("Failed to perform ES search on index %s, query was: <%s>, error was: %+v", index, query, err)
		return 0, values, waitIndex, errors.Wrapf(err, "Failed to perform ES search on index %s, query was: <%s>, error was: %+v", index, query, err)
	}
	defer res.Body.Close()

	if res.IsError() {
		var e map[string]interface{}
		if err := json.NewDecoder(res.Body).Decode(&e); err != nil {
			log.Printf("An error occurred while performing ES search on index %s, query was: <%s>, response code was %d (%s). Wasn't able to decode response body !", index, query, res.StatusCode, res.Status())
			return 0, values, waitIndex, errors.Wrapf(err, "An error occurred while performing ES search on index %s, query was: <%s>, response code was %d (%s). Wasn't able to decode response body !", index, query, res.StatusCode, res.Status())
		}
		log.Printf("An error occurred while performing ES search on index %s, query was: <%s>, response code was %d (%s). Response body was: %+v", index, query, res.StatusCode, res.Status(), e)
		return 0, values, waitIndex, errors.Wrapf(err, "An error occurred while performing ES search on index %s, query was: <%s>, response code was %d (%s). Response body was: %+v", index, query, res.StatusCode, res.Status(), e)
	}

	var r map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		log.Printf("Not able to decode ES response while performing ES search on index %s, query was: <%s>, response code was %d (%s)", index, query, res.StatusCode, res.Status())
		return 0, values, waitIndex, errors.Wrapf(err, "Not able to decode ES response while performing ES search on index %s, query was: <%s>, response code was %d (%s)", index, query, res.StatusCode, res.Status())
	}

	hits := int(r["hits"].(map[string]interface{})["total"].(float64))
	duration := int(r["took"].(float64))
	log.Debugf("Search ES request on index %s took %dms, hits=%d, response code was %d (%s)", index, duration, hits, res.StatusCode, res.Status())

	var lastIndex = waitIndex

	// Print the ID and document source for each hit.
	for _, hit := range r["hits"].(map[string]interface{})["hits"].([]interface{}) {
		id := hit.(map[string]interface{})["_id"].(string)
		source := hit.(map[string]interface{})["_source"].(map[string]interface{})
		iid := source["iid"]
		iidInt64, err := parseInt64StringToUint64(iid.(string))
		if err != nil {
			log.Printf("Not able to parse iid_str property %s as uint64, document id: %s, source: %+v, ignoring this document !", iid, id, source)
		} else {
			jsonString, err := json.Marshal(source)
			if err != nil {
				log.Printf("Not able to marshall document source, document id: %s, source: %+v, ignoring this document !", id, source)
			} else {
				// since the result is sorted on iid, we can use the last hit to define lastIndex
				lastIndex = iidInt64
				// append value to result
				values = append(values, store.KeyValueOut{
					Key:             id,
					LastModifyIndex: iidInt64,
					Value:           source,
					RawValue:        jsonString,
				})
			}
		}
	}

	log.Debugf("doQueryEs called result waitIndex: %d, LastIndex: %d, len(values): %d", waitIndex, lastIndex, len(values))
	return hits, values, lastIndex, nil
}

func sendBulkRequest(c *elasticsearch6.Client, opeCount int, body *[]byte) error {
	log.Printf("About to bulk request containing %d operations (%d bytes)", opeCount, len(*body))
	if log.IsDebug() {
		log.Debugf("About to send bulk request query to ES: %s", string(*body))
	}

	// Prepare ES bulk request
	req := esapi.BulkRequest{
		Body: bytes.NewReader(*body),
	}
	res, err := req.Do(context.Background(), c)

	defer res.Body.Close()

	if err != nil {
		return err
	} else if res.IsError() {
		return errors.Errorf("Error while sending bulk request, response code was <%d> and response message was <%s>", res.StatusCode, res.String())
	} else {
		var rsp map[string]interface{}
		json.NewDecoder(res.Body).Decode(&rsp)
		if rsp["errors"].(bool) {
			// The bulk request contains errors
			return errors.Errorf("The bulk request succeeded, but the response contains errors : %+v", rsp)
		}
	}
	log.Printf("Bulk request containing %d operations (%d bytes) has been accepted without errors", opeCount, len(*body))
	return nil
}
