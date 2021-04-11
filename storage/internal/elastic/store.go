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
	"math"
	"strings"
	"time"
)

type elasticStore struct {
	codec    encoding.Codec
	esClient *elasticsearch6.Client
	cfg      elasticStoreConf
}

// NewStore returns a new Elastic store.
// Since the elastic store can only manage logs or events, it will panic is it's configured for anything else.
// At init stage, we display ES cluster info and initialise indexes if they are not found.
func NewStore(cfg config.Configuration, storeConfig config.Store) (store.Store, error) {

	// Just fail if this storage is used for anything different from logs or events
	for _, t := range storeConfig.Types {
		if t != "Log" && t != "Event" {
			return nil, errors.Errorf("Elastic store is not able to manage <%s>, just Log or Event, please change your config", t)
		}
	}

	// Get specific config from storage properties
	elasticStoreConfig, err := getElasticStoreConfig(cfg, storeConfig)
	if err != nil {
		return nil, err
	}

	esClient, err := prepareEsClient(elasticStoreConfig)
	if err != nil {
		return nil, err
	}

	err = initStorageIndex(esClient, elasticStoreConfig, "logs")
	if err != nil {
		return nil, errors.Wrapf(err, "Not able to init index for eventType <%s>", "logs")
	}
	err = initStorageIndex(esClient, elasticStoreConfig, "events")
	if err != nil {
		return nil, errors.Wrapf(err, "Not able to init index for eventType <%s>", "events")
	}

	return &elasticStore{encoding.JSON, esClient, elasticStoreConfig}, nil
}

// Set index a document (log or event) into ES.
func (s *elasticStore) Set(ctx context.Context, k string, v interface{}) error {
	log.Debugf("Set called will key %s", k)

	if err := utils.CheckKeyAndValue(k, v); err != nil {
		return err
	}

	storeType, body, err := buildElasticDocument(k, v)
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
		DocumentType: "_doc",
		Body:         bytes.NewReader(body),
	}
	res, err := req.Do(context.Background(), s.esClient)
	defer closeResponseBody("IndexRequest:"+indexName, res)
	if err != nil || res.IsError() {
		err = handleESResponseError(res, "Index:"+indexName, string(body), err)
		return err
	}
	return nil
}

// SetCollection index collections using ES bulk requests.
// We consider both 'max_bulk_size' and 'max_bulk_count' to define bulk requests size.
func (s *elasticStore) SetCollection(ctx context.Context, keyValues []store.KeyValueIn) error {
	totalDocumentCount := len(keyValues)
	log.Printf("SetCollection called with an array of size %d", totalDocumentCount)
	start := time.Now()

	if keyValues == nil || totalDocumentCount == 0 {
		return nil
	}

	// Just estimate the iteration count
	iterationCount := int(math.Ceil(float64(totalDocumentCount) / float64(s.cfg.maxBulkCount)))
	log.Printf(
		"max_bulk_count is %d, so a minimum of %d iterations will be necessary to bulk index the %d documents",
		s.cfg.maxBulkCount, iterationCount, totalDocumentCount,
	)

	// The current index in []keyValues (also the number of documents indexed)
	var kvi = 0
	// The number of iterations
	var i = 0
	// Iterate over the []keyValues
	for {
		if kvi == totalDocumentCount {
			// We have reached the end of []keyValues
			break
		}
		fmt.Printf("Bulk iteration %d", i)

		maxBulkSizeInBytes := s.cfg.maxBulkSize * 1024
		// Prepare a slice of max capacity
		var body = make([]byte, 0, maxBulkSizeInBytes)
		// Number of operation in the current bulk request
		opeCount := 0
		// Each iteration is a single bulk request
		for {
			if kvi == totalDocumentCount || opeCount == s.cfg.maxBulkCount {
				// We have reached the end of []keyValues OR the max items allowed in a single bulk request (max_bulk_count)
				break
			}
			added, err := eventuallyAppendValueToBulkRequest(s.cfg, &body, keyValues[kvi], maxBulkSizeInBytes)
			if err != nil {
				return err
			} else if !added {
				// The document hasn't been added (too big), let's include it in next bulk
				break
			} else {
				kvi++
				opeCount++
			}
		}
		// The bulk request must be terminated by a newline
		body = append(body, "\n"...)
		// Send the request
		err := sendBulkRequest(s.esClient, opeCount, &body)
		if err != nil {
			return err
		}
		// Increment the number of iterations
		i++
	}
	elapsed := time.Since(start)
	log.Printf("A total of %d documents have been successfully indexed using %d bulk requests, took %v", kvi, i, elapsed)
	return nil
}

// Delete removes ES documents using a deleteByRequest query.
func (s *elasticStore) Delete(ctx context.Context, k string, recursive bool) error {
	log.Debugf("Delete called k: %s, recursive: %t", k, recursive)

	// Extract index name and deploymentID by parsing the key
	storeType, deploymentID := extractStoreTypeAndDeploymentID(k)
	indexName := getIndexName(s.cfg, storeType)
	log.Debugf("storeType is: %s, indexName is %s, deploymentID is: %s", storeType, indexName, deploymentID)

	query := `{"query" : { "term": { "deploymentId" : "` + deploymentID + `" }}}`
	log.Debugf("query is : %s", query)

	var MaxInt = 1024000

	req := esapi.DeleteByQueryRequest{
		Index:     []string{indexName},
		Size:      &MaxInt,
		Body:      strings.NewReader(query),
		Conflicts: "proceed",
	}
	res, err := req.Do(context.Background(), s.esClient)
	defer closeResponseBody("DeleteByQueryRequest:"+indexName, res)
	err = handleESResponseError(res, "DeleteByQueryRequest:"+indexName, query, err)
	return err
}

// GetLastModifyIndex return the last index which is found by querying ES using aggregation and a 0 size request.
func (s *elasticStore) GetLastModifyIndex(k string) (lastIndex uint64, e error) {
	log.Debugf("GetLastModifyIndex called k: %s", k)

	// Extract index name and deploymentID by parsing the key
	storeType, deploymentID := extractStoreTypeAndDeploymentID(k)
	indexName := getIndexName(s.cfg, storeType)
	log.Debugf("storeType is: %s, indexName is: %s, deploymentID is: %s", storeType, indexName, deploymentID)

	// The lastIndex is query by using ES aggregation query ~= MAX(iid) HAVING deploymentId
	query := buildLastModifiedIndexQuery(deploymentID)
	log.Debugf("buildLastModifiedIndexQuery is : %s", query)

	resSearch, err := s.esClient.Search(
		s.esClient.Search.WithContext(context.Background()),
		s.esClient.Search.WithIndex(indexName),
		s.esClient.Search.WithSize(0),
		s.esClient.Search.WithBody(strings.NewReader(query)),
	)
	defer closeResponseBody("LastModifiedIndexQuery for "+k, resSearch)
	e = handleESResponseError(resSearch, "LastModifiedIndexQuery for "+k, query, err)
	if e != nil {
		return
	}

	var r map[string]interface{}
	if err := json.NewDecoder(resSearch.Body).Decode(&r); err != nil {
		e = errors.Wrapf(
			err,
			"Not able to parse response body after LastModifiedIndexQuery was sent for key %s, status was %s, query was: %s",
			k, resSearch.Status(), query,
		)
		return
	}

	total := r["hits"].(map[string]interface{})["total"].(float64)
	if total > 0 {
		// ES returns aggregations as float, we have a precision loss of few ns
		lastIndexR := r["aggregations"].(map[string]interface{})["max_iid"].(map[string]interface{})["last_index"].(map[string]interface{})["value"].(float64)
		log.Debugf("Received lastIndexReceived: %v, lastIndex: %v", lastIndexR, lastIndex)
		lastIndex = uint64(lastIndexR)
		// The ES max result was a float, there is a risk that this is not really the lastIndex
		// We need to verify
		lastIndex = s.verifyLastIndex(indexName, deploymentID, lastIndex)
	}
	return lastIndex, nil
}

// We need to ensure the lastIndex returned by the aggregation query is really the last
// Actually, when elasticsearch aggregates, it returns a float so we loss precession (few ns).
// We request the docs with iid > waitIndex to ensure the returned lastIndex is REALLY the last.
func (s *elasticStore) verifyLastIndex(indexName string, deploymentID string, estimatedLastIndex uint64) uint64 {
	query := getListQuery(deploymentID, estimatedLastIndex, 0)
	// size = 1 no need for the documents
	hits, _, lastIndex, err := doQueryEs(context.Background(), s.esClient, s.cfg, indexName, query, estimatedLastIndex, 1, "desc")
	if err != nil {
		log.Printf("An error occurred while verifying lastIndex, returning the initial value %d, error was : %+v",
			estimatedLastIndex, err)
	}
	log.Printf("%d hits while searching %s (%s) using the estimated lastIndex %d, lastIndex is now %d",
		hits, indexName, deploymentID, estimatedLastIndex, lastIndex)
	return lastIndex
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

	query := getListQuery(deploymentID, waitIndex, 0)

	now := time.Now()
	end := now.Add(timeout - s.cfg.esRefreshWaitTimeout)
	log.Debugf("Now is : %v, date after timeout will be %v (ES timeout duration will be %v)", now, end, timeout-s.cfg.esRefreshWaitTimeout)
	var values = make([]store.KeyValueOut, 0)
	var lastIndex = waitIndex
	var hits = 0
	var err error
	for {
		// first just query to know if they is something to fetch, we just want the max iid (so order desc, size 1)
		hits, values, lastIndex, err = doQueryEs(ctx, s.esClient, s.cfg, indexName, query, waitIndex, 1, "desc")
		if err != nil {
			return values, waitIndex, errors.Wrapf(err, "Failed to request ES logs or events, error was: %+v", err)
		}
		now := time.Now()
		if hits > 0 || now.After(end) {
			break
		}

		log.Debugf("hits is %d and timeout not reached, sleeping %v ...", hits, s.cfg.esQueryPeriod)
		select {
		case <-time.After(s.cfg.esQueryPeriod):
			continue
		case <-ctx.Done():
			return values, lastIndex, nil
		}
	}
	if hits > 0 {
		// we do have something to retrieve, we will just wait esRefreshWaitTimeout to let any document that has just been stored to be indexed
		// then we just retrieve this 'time window' (between waitIndex and lastIndex)
		query := getListQuery(deploymentID, waitIndex, lastIndex)
		if s.cfg.esForceRefresh {
			// force refresh for this index
			refreshIndex(s.esClient, indexName)
		}
		time.Sleep(s.cfg.esRefreshWaitTimeout)
		oldHits := hits
		hits, values, lastIndex, err = doQueryEs(ctx, s.esClient, s.cfg, indexName, query, waitIndex, 10000, "asc")
		if err != nil {
			return values, waitIndex, errors.Wrapf(err, "Failed to request ES logs or events (after waiting for refresh)")
		}
		if log.IsDebug() && hits > oldHits {
			log.Debugf("%d > %d so sleeping %v to wait for ES refresh was useful (index %s), %d documents has been fetched",
				hits, oldHits, s.cfg.esRefreshWaitTimeout, indexName, len(values),
			)
		}
	}
	log.Debugf("List called result k: %s, waitIndex: %d, timeout: %v, LastIndex: %d, len(values): %d",
		k, waitIndex, timeout, lastIndex, len(values))
	return values, lastIndex, err
}

// Get is not used for logs nor events: fails in FATAL.
func (s *elasticStore) Get(k string, v interface{}) (bool, error) {
	if err := utils.CheckKeyAndValue(k, v); err != nil {
		return false, err
	}
	return false, errors.Errorf("Function Get(string, interface{}) not yet implemented for Elastic store !")
}

// Exist is not used for logs nor events: fails in FATAL.
func (s *elasticStore) Exist(k string) (bool, error) {
	if err := utils.CheckKey(k); err != nil {
		return false, err
	}
	return false, errors.Errorf("Function Exist(string) not yet implemented for Elastic store !")
}

// Keys is not used for logs nor events: fails in FATAL.
func (s *elasticStore) Keys(k string) ([]string, error) {
	return nil, errors.Errorf("Function Keys(string) not yet implemented for Elastic store !")
}
