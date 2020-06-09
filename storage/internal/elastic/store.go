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

// Package elastic provides an implementation of a storage that index/get documents to/from Elasticsearch 6.x.
// This store can only manage logs and events for the moment. It will fail if you try to use it for other store types.
package elastic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	elasticsearch6 "github.com/elastic/go-elasticsearch/v6"
	"github.com/elastic/go-elasticsearch/v6/esapi"
	"github.com/pkg/errors"
	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/log"
	"github.com/ystia/yorc/v4/storage/encoding"
	"github.com/ystia/yorc/v4/storage/store"
	"github.com/ystia/yorc/v4/storage/utils"
	"io/ioutil"
	"math"
	"strings"
	"time"
)

type elasticStore struct {
	codec     encoding.Codec
	esClient  *elasticsearch6.Client
	clusterID string
	cfg       elasticStoreConf
}

// NewStore returns a new Elastic store.
// Since the elastic store can only manage logs or events, it will panic is it's configured for anything else.
// At init stage, we display ES cluster info and initialise indexes if they are not found.
func NewStore(cfg config.Configuration, storeConfig config.Store) store.Store {

	// Just fail if this storage is used for anything different from logs or events
	for _, t := range storeConfig.Types {
		if t != "Log" && t != "Event" {
			log.Panicf("Elastic store is not able to manage <%s>, just Log or Event, please change your config", t)
		}
	}

	// Get specific config from storage properties
	esCfg := getElasticStoreConfig(storeConfig)

	var esConfig elasticsearch6.Config
	esConfig = elasticsearch6.Config{
		Addresses: esCfg.esUrls,
	}

	if len(esCfg.caCertPath) > 0 {
		log.Printf("Reading CACert file from %s", esCfg.caCertPath)
		cert, err := ioutil.ReadFile(esCfg.caCertPath)
		if err != nil {
			log.Panicf("Not able to read Cert file from <%s>, error was %+v", esCfg.caCertPath, err)
		}
		esConfig.CACert = cert
	}

	log.Printf("Elastic storage will run using this configuration: %+v", esCfg)
	log.Printf("\t- Index prefix will be %s", esCfg.indicePrefix)
	log.Printf("\t- Will query ES for logs or events every %v and will wait for index refresh during %v", esCfg.esQueryPeriod, esCfg.esRefreshWaitTimeout)
	willRefresh := ""
	if !esCfg.esForceRefresh {
		willRefresh = "not "
	}
	log.Printf("\t- Will %srefresh index before waiting for indexation", willRefresh)
	log.Printf("\t- While migrating data, the max bulk request size will be %d documents and will never exceed %d kB", esCfg.maxBulkCount, esCfg.maxBulkSize)
	log.Printf("\t- Will use this ES client configuration: %+v", esConfig)

	esClient, _ := elasticsearch6.NewClient(esConfig)

	log.Printf("Here is the ES cluster info")
	log.Println(esClient.Info())
	log.Printf("ClusterID: %s, ServerID: %s", cfg.ClusterID, cfg.ServerID)
	var clusterID = cfg.ClusterID
	if len(clusterID) == 0 {
		clusterID = cfg.ServerID
	}

	initStorageIndex(esClient, esCfg.indicePrefix+"logs")
	initStorageIndex(esClient, esCfg.indicePrefix+"events")
	debugIndexSetting(esClient, esCfg.indicePrefix+"logs")
	debugIndexSetting(esClient, esCfg.indicePrefix+"events")
	return &elasticStore{encoding.JSON, esClient, clusterID, esCfg}
}

// Set index a document (log or event) into ES.
func (s *elasticStore) Set(ctx context.Context, k string, v interface{}) error {
	log.Debugf("Set called will key %s", k)

	if err := utils.CheckKeyAndValue(k, v); err != nil {
		return err
	}

	storeType, body, err := buildElasticDocument(s.clusterID, k, v)
	if err != nil {
		return err
	}

	indexName := getIndexName(s.cfg, storeType)
	if log.IsDebug() {
		log.Debugf("About to index this document into ES index <%s> : %+v", indexName, string(body))
	}

	// Prepare ES request
	req := esapi.IndexRequest{
		Index:        indexName,
		DocumentType: "logs_or_event",
		Body:         bytes.NewReader(body),
	}
	res, err := req.Do(context.Background(), s.esClient)
	debugESResponse("IndexRequest:"+indexName, res, err)

	defer res.Body.Close()

	if err != nil {
		return err
	} else if res.IsError() {
		return errors.Errorf("Error while indexing document, response code was <%d> and response message was <%s>", res.StatusCode, res.String())
	} else {
		return nil
	}
}

// SetCollection index collections using ES bulk requests.
// We consider both 'max_bulk_size' and 'max_bulk_count' to define bulk requests size.
func (s *elasticStore) SetCollection(ctx context.Context, keyValues []store.KeyValueIn) error {
	totalDocumentCount := len(keyValues)
	log.Printf("SetCollection called with an array of size %d", totalDocumentCount)

	if keyValues == nil || totalDocumentCount == 0 {
		return nil
	}

	iterationCount := int(math.Ceil(float64(totalDocumentCount) / float64(s.cfg.maxBulkCount)))
	log.Printf("max_bulk_count is %d, so a minimum of %d iterations will be necessary to bulk index the %d documents", s.cfg.maxBulkCount, iterationCount, totalDocumentCount)

	// The current index in []keyValues (also the number of documents indexed)
	var kvi = 0
	// The number of iterations
	var i = 0
	for {
		if kvi == totalDocumentCount {
			break
		}
		fmt.Printf("Bulk iteration %d", i)

		maxBulkSizeInBytes := s.cfg.maxBulkSize * 1024
		var body = make([]byte, 0, maxBulkSizeInBytes)
		bulkActionCount := 0
		for {
			if kvi == totalDocumentCount || bulkActionCount == s.cfg.maxBulkCount {
				break
			}
			added, err := eventuallyAppendValueToBulkRequest(s.cfg, s.clusterID, &body, keyValues[kvi], maxBulkSizeInBytes)
			if err != nil {
				return err
			} else if !added {
				break
			} else {
				kvi++
				bulkActionCount++
			}
		}

		// The bulk request must be terminated by a newline
		body = append(body, "\n"...)

		log.Printf("About to bulk request index using %d documents (%d bytes)", bulkActionCount, len(body))
		if log.IsDebug() {
			log.Debugf("About to send bulk request query to ES: %s", string(body))
		}

		// Prepare ES bulk request
		req := esapi.BulkRequest{
			Body: bytes.NewReader(body),
		}
		res, err := req.Do(context.Background(), s.esClient)

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
			log.Printf("Bulk request containing %d documents (%d bytes) has been accepted without errors", bulkActionCount, len(body))
		}
		// increment the number of iterations
		i++
	}
	log.Printf("A total of %d documents have been successfully indexed using %d bulk requests", kvi, i)

	return nil
}

// Delete removes ES documents using a deleteByRequest query.
func (s *elasticStore) Delete(ctx context.Context, k string, recursive bool) error {
	log.Debugf("Delete called k: %s, recursive: %t", k, recursive)

	// Extract indice name and deploymentID by parsing the key
	storeType, deploymentID := extractStoreTypeAndDeploymentID(k)
	indexName := getIndexName(s.cfg, storeType)
	log.Debugf("storeType is: %s, indexName is %s, deploymentID is: %s", storeType, indexName, deploymentID)

	query := `{"query" : { "bool" : { "must" : [{ "term": { "clusterId" : "` + s.clusterID + `" }}, { "term": { "deploymentId" : "` + deploymentID + `" }}]}}}`
	log.Debugf("query is : %s", query)

	var MaxInt = 1024000

	req := esapi.DeleteByQueryRequest{
		Index: []string{indexName},
		Size:  &MaxInt,
		Body:  strings.NewReader(query),
	}
	res, err := req.Do(context.Background(), s.esClient)
	debugESResponse("DeleteByQueryRequest:"+indexName, res, err)

	defer res.Body.Close()
	return err
}

// GetLastModifyIndex return the last index which is found by querying ES using aggregation and a 0 size request.
func (s *elasticStore) GetLastModifyIndex(k string) (uint64, error) {
	log.Debugf("GetLastModifyIndex called k: %s", k)

	// Extract indice name and deploymentID by parsing the key
	storeType, deploymentID := extractStoreTypeAndDeploymentID(k)
	indexName := getIndexName(s.cfg, storeType)
	log.Debugf("storeType is: %s, indexName is: %s, deploymentID is: %s", storeType, indexName, deploymentID)

	// The lastIndex is query by using ES aggregation query ~= MAX(iid) HAVING deploymentId AND clusterId
	query := buildLastModifiedIndexQuery(s.clusterID, deploymentID)
	log.Debugf("buildLastModifiedIndexQuery is : %s", query)

	res, err := s.esClient.Search(
		s.esClient.Search.WithContext(context.Background()),
		s.esClient.Search.WithIndex(indexName),
		s.esClient.Search.WithSize(0),
		s.esClient.Search.WithBody(strings.NewReader(query)),
	)
	if err != nil {
		log.Printf("ERROR: %s", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		var e map[string]interface{}
		if err := json.NewDecoder(res.Body).Decode(&e); err != nil {
			log.Printf("error parsing the response body: %s", err)
		} else {
			// Print the response status and error information.
			log.Printf("[%s] %s: %s",
				res.Status(),
				e["error"].(map[string]interface{})["type"],
				e["error"].(map[string]interface{})["reason"],
			)
		}
	}

	var r lastIndexResponse
	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		log.Printf("Error parsing the response body: %s", err)
		return 0, err
	}

	hits := r.hits.total
	var lastIndex uint64 = 0
	if hits > 0 {
		lastIndex, err = parseInt64StringToUint64(r.aggregations.logsOrEvents.lastIndex.value)
		if err != nil {
			return lastIndex, err
		}
	}

	// Print the response status, number of results, and request duration.
	log.Debugf(
		"[%s] %d hits; lastIndex: %d",
		res.Status(),
		hits,
		lastIndex,
	)

	return lastIndex, nil

}

// List simulates long polling request by :
// - periodically querying ES for documents (Aggregation to get the max iid and 0 size result).
// - if a some result is found, wait some time (es_refresh_wait_timeout) in order to:
//   	- let ES index recently added documents AND to let
// 		- let Yorc eventually Set a document that has a less iid than the older known document in ES (concurrence issues)
// - if no result if found after the the given 'timeout', return empty slice
func (s *elasticStore) List(ctx context.Context, k string, waitIndex uint64, timeout time.Duration) ([]store.KeyValueOut, uint64, error) {
	log.Debugf("List called k: %s, waitIndex: %d, timeout: %v", k, waitIndex, timeout)
	if err := utils.CheckKey(k); err != nil {
		return nil, 0, err
	}

	// Extract indice name by parsing the key
	storeType, deploymentID := extractStoreTypeAndDeploymentID(k)
	indexName := getIndexName(s.cfg, storeType)
	log.Debugf("storeType is: %s, indexName is: %s, deploymentID is: %s", storeType, indexName, deploymentID)

	query := getListQuery(s.clusterID, deploymentID, waitIndex, 0)

	now := time.Now()
	end := now.Add(timeout - s.cfg.esRefreshWaitTimeout)
	log.Debugf("Now is : %v, date after timeout will be %v (ES timeout duration will be %v)", now, end, timeout-s.cfg.esRefreshWaitTimeout)
	var values = make([]store.KeyValueOut, 0)
	var lastIndex = waitIndex
	var hits = 0
	var err error
	for {
		// first just query to know if they is something to fetch, we just want the max iid (so order desc, size 1)
		hits, values, lastIndex, err = doQueryEs(s.esClient, indexName, query, waitIndex, 1, "desc")
		if err != nil {
			return values, waitIndex, errors.Wrapf(err, "Failed to request ES logs or events, error was: %+v", err)
		}
		now := time.Now()
		if hits > 0 || now.After(end) {
			break
		}
		log.Debugf("hits is %d and timeout not reached, sleeping %v ...", hits, s.cfg.esQueryPeriod)
		time.Sleep(s.cfg.esQueryPeriod)
	}
	if hits > 0 {
		// we do have something to retrieve, we will just wait esRefreshWaitTimeout to let any document that has just been stored to be indexed
		// then we just retrieve this 'time window' (between waitIndex and lastIndex)
		query := getListQuery(s.clusterID, deploymentID, waitIndex, lastIndex)
		if s.cfg.esForceRefresh {
			// force refresh for this index
			refreshIndex(s.esClient, indexName)
		}
		time.Sleep(s.cfg.esRefreshWaitTimeout)
		oldHits := hits
		hits, values, lastIndex, err = doQueryEs(s.esClient, indexName, query, waitIndex, 10000, "asc")
		if err != nil {
			return values, waitIndex, errors.Wrapf(err, "Failed to request ES logs or events (after waiting for refresh), error was: %+v", err)
		}
		if log.IsDebug() && hits > oldHits {
			log.Debugf("%d > %d so sleeping %v to wait for ES refresh was usefull (index %s), %d documents has been fetched", hits, oldHits, s.cfg.esRefreshWaitTimeout, indexName, len(values))
		}
	}
	log.Debugf("List called result k: %s, waitIndex: %d, timeout: %v, LastIndex: %d, len(values): %d", k, waitIndex, timeout, lastIndex, len(values))
	return values, lastIndex, err
}

// Get is not used for logs nor events: fails in FATAL.
func (s *elasticStore) Get(k string, v interface{}) (bool, error) {
	if err := utils.CheckKeyAndValue(k, v); err != nil {
		return false, err
	}
	log.Fatalf("Function Get(string, interface{}) not yet implemented for Elastic store !")
	return false, errors.Errorf("Function Get(string, interface{}) not yet implemented for Elastic store !")
}

// Exist is not used for logs nor events: fails in FATAL.
func (s *elasticStore) Exist(k string) (bool, error) {
	if err := utils.CheckKey(k); err != nil {
		return false, err
	}
	log.Fatalf("Function Exist(string) not yet implemented for Elastic store !")
	return false, errors.Errorf("Function Exist(string) not yet implemented for Elastic store !")
}

// Keys is not used for logs nor events: fails in FATAL.
func (s *elasticStore) Keys(k string) ([]string, error) {
	log.Fatalf("Function Keys(string) not yet implemented for Elastic store !")
	return nil, errors.Errorf("Function Keys(string) not yet implemented for Elastic store !")
}
