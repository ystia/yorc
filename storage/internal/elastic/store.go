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
	"context"
	"github.com/pkg/errors"
	"github.com/ystia/yorc/v4/storage/encoding"
	"github.com/ystia/yorc/v4/storage/store"
	"github.com/ystia/yorc/v4/storage/utils"
	"time"
	elasticsearch6 "github.com/elastic/go-elasticsearch/v6"
	"github.com/ystia/yorc/v4/log"
	"regexp"
	"github.com/elastic/go-elasticsearch/v6/esapi"
	"encoding/json"
	"bytes"
	"github.com/ystia/yorc/v4/config"
	"strings"
	"strconv"
	"math"
	"fmt"
	"reflect"
	"github.com/spf13/cast"
	"io/ioutil"
)

// elasticStoreConf represents the elastic store configuration that can be set in store.properties configuration.
type elasticStoreConf struct {
	// The ES cluster urls (array or CSV)
	esUrls []string `json:"es_urls"`
	// The path to the CACert file when TLS is activated for ES
	caCertPath string `json:"ca_cert_path"`
	// All index used by yorc will be prefixed by this prefix
	indicePrefix string `json:"index_prefix" default:"yorc_"`
	// When querying logs and event, we wait this timeout before each request when it returns nothing
	// (until something is returned or the waitTimeout is reached)
	esQueryPeriod time.Duration `json:"es_query_period" default:"5s"`
	// This timeout is used to wait for more than refresh_interval = 1s when querying logs and events indexes
	esRefreshWaitTimeout time.Duration `json:"es_refresh_wait_timeout" default:"5s"`
	// When querying ES, force refresh index before waiting for refresh
	esForceRefresh bool `json:"es_force_refresh" default:"true"`
	// This is the maximum size (in kB) of bulk request sent while migrating data
	maxBulkSize int `json:"max_bulk_size" default:"4000"`
	// This is the maximum size (in term of number of documents) of bulk request sent while migrating data
	maxBulkCount int `json:"max_bulk_count" default:"1000"`
}

var pfalse = false

type elasticStore struct {
	codec     encoding.Codec
	esClient  *elasticsearch6.Client
	clusterId string
	cfg       elasticStoreConf
}

// structs for lastIndexRequest response decoding.
type lastIndexResponse struct {
	hits         hits                  `json:"hits"`
	aggregations logOrEventAggregation `json:"aggregations"`
}
type hits struct {
	total int `json:"total"`
}
type logOrEventAggregation struct {
	logs_or_events lastIndexAggregation `json:"logs_or_events"`
}
type lastIndexAggregation struct {
	doc_count  int         `json:"doc_count"`
	last_index stringValue `json:"last_index"`
}
type stringValue struct {
	value string `json:"value"`
}

// Precompiled regex to extract storeType and timestamp from a key of the form: "_yorc/logs/MyApp/2020-06-07T21:03:17.812178429Z".
var storeTypeAndTimestampRegex = regexp.MustCompile(`(?m)\_yorc\/(\w+)\/.+\/(.*)`)

// Parse a key of form  "_yorc/logs/MyApp/2020-06-07T21:03:17.812178429Z" to get the store type (logs|events) and the timestamp.
func extractStoreTypeAndTimestamp(k string) (storeType string, timestamp string) {
	res := storeTypeAndTimestampRegex.FindAllStringSubmatch(k, -1)
	for i := range res {
		storeType = res[i][1]
		timestamp = res[i][2]
	}
	return storeType, timestamp
}

// Precompiled regex to extract storeType and deploymentId from a key of the form: "_yorc/events/" or "_yorc/logs/MyApp" or "_yorc/logs/MyApp/".
var storeTypeAndDeploymentIdRegex = regexp.MustCompile(`(?m)\_yorc\/(\w+)\/?(.+)?\/?`)

// Parse a key of form "_yorc/events/" or "_yorc/logs/MyApp" or "_yorc/logs/MyApp/" to get the store type (logs|events) and eventually the deploymentId.
func extractStoreTypeAndDeploymentId(k string) (storeType string, deploymentId string) {
	res := storeTypeAndDeploymentIdRegex.FindAllStringSubmatch(k, -1)
	for i := range res {
		storeType = res[i][1]
		if len(res[i]) == 3 {
			deploymentId = res[i][2]
			if strings.HasSuffix(deploymentId, "/") {
				deploymentId = deploymentId[:len(deploymentId)-1]
			}
		}
	}
	return storeType, deploymentId
}

// Get the JSON tag for this field (for internal usage only !).
func getElasticStorageConfigPropertyTag(fn string, tn string) string {
	t := reflect.TypeOf(elasticStoreConf{})
	f, found := t.FieldByName(fn)
	if !found {
		log.Fatalf("Not able to get field %s on elasticStoreConf struct, there is an issue with this code !", fn)
	}
	tv := f.Tag.Get(tn)
	if tv == "" {
		log.Fatalf("Not able to get field %s's tag %s value on elasticStoreConf struct, there is an issue with this code !", fn, tn)
	}
	return tv
}

// The configuration of the elastic store is defined regarding default values and store 'properties' dynamic map.
// Will fail if the required es_urls is not set in store 'properties'
func getElasticStoreConfig(storeConfig config.Store) elasticStoreConf {
	var cfg elasticStoreConf

	storeProperties := storeConfig.Properties

	// The ES urls is required
	k := getElasticStorageConfigPropertyTag("esUrls", "json")
	if storeProperties.IsSet(k) {
		cfg.esUrls = storeProperties.GetStringSlice(k)
		if cfg.esUrls == nil || len(cfg.esUrls) == 0 {
			log.Fatalf("Not able to get ES configuration for elastic store, es_urls store property seems empty : %+v", storeProperties.Get(k))
		}
	} else {
		log.Fatal("Not able to get ES configuration for elastic store, es_urls store property should be set !")
	}

	// Define store optional / default configuration
	k = getElasticStorageConfigPropertyTag("caCertPath", "json")
	if storeProperties.IsSet(k) {
		cfg.caCertPath = storeProperties.GetString(k)
	}
	k = getElasticStorageConfigPropertyTag("esQueryPeriod", "json")
	if storeProperties.IsSet(k) {
		cfg.esQueryPeriod = storeProperties.GetDuration(k)
	} else {
		cfg.esQueryPeriod = cast.ToDuration(getElasticStorageConfigPropertyTag("esQueryPeriod", "default"))
	}
	k = getElasticStorageConfigPropertyTag("esForceRefresh", "json")
	if storeProperties.IsSet(k) {
		cfg.esForceRefresh = storeProperties.GetBool(k)
	} else {
		cfg.esForceRefresh = cast.ToBool(getElasticStorageConfigPropertyTag("esForceRefresh", "default"))
	}
	k = getElasticStorageConfigPropertyTag("esRefreshWaitTimeout", "json")
	if storeProperties.IsSet(k) {
		cfg.esRefreshWaitTimeout = storeProperties.GetDuration(k)
	} else {
		cfg.esRefreshWaitTimeout = cast.ToDuration(getElasticStorageConfigPropertyTag("esRefreshWaitTimeout", "default"))
	}
	k = getElasticStorageConfigPropertyTag("indicePrefix", "json")
	if storeProperties.IsSet(k) {
		cfg.indicePrefix = storeProperties.GetString(k)
	} else {
		cfg.indicePrefix = getElasticStorageConfigPropertyTag("indicePrefix", "default")
	}
	k = getElasticStorageConfigPropertyTag("maxBulkSize", "json")
	if storeProperties.IsSet(k) {
		cfg.maxBulkSize = storeProperties.GetInt(k)
	} else {
		cfg.maxBulkSize = cast.ToInt(getElasticStorageConfigPropertyTag("maxBulkSize", "default"))
	}
	k = getElasticStorageConfigPropertyTag("maxBulkCount", "json")
	if storeProperties.IsSet(k) {
		cfg.maxBulkCount = storeProperties.GetInt(k)
	} else {
		cfg.maxBulkCount = cast.ToInt(getElasticStorageConfigPropertyTag("maxBulkCount", "default"))
	}

	return cfg
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
	log.Println(esClient.Info());
	log.Printf("ClusterID: %s, ServerID: %s", cfg.ClusterID, cfg.ServerID)
	var clusterId string = cfg.ClusterID
	if len(clusterId) == 0 {
		clusterId = cfg.ServerID
	}

	initStorageIndices(esClient, esCfg.indicePrefix+"logs")
	initStorageIndices(esClient, esCfg.indicePrefix+"events")
	debugIndexSetting(esClient, esCfg.indicePrefix+"logs")
	debugIndexSetting(esClient, esCfg.indicePrefix+"events")
	return &elasticStore{encoding.JSON, esClient, clusterId, esCfg}
}

// Init ES index for logs or events storage: create it if not found.
func initStorageIndices(esClient *elasticsearch6.Client, indexName string) {

	log.Printf("Checking if index <%s> already exists", indexName)

	// check if the sequences index exists
	req := esapi.IndicesExistsRequest{
		Index:           []string{indexName},
		ExpandWildcards: "none",
		AllowNoIndices:  &pfalse,
	}
	res, err := req.Do(context.Background(), esClient)
	debugESResponse("IndicesExistsRequest:"+indexName, res, err)
	defer res.Body.Close()
	log.Printf("Status Code for IndicesExistsRequest (%s): %d", indexName, res.StatusCode)

	if res.StatusCode == 200 {
		log.Printf("Indice %s was found, nothing to do !", indexName)
	}

	if res.StatusCode == 404 {
		log.Printf("Indice %s was not found, let's create it !", indexName)

		requestBodyData := `
{
     "settings": {
         "refresh_interval": "1s"
     },
     "mappings": {
         "logs_or_event": {
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
                     "type": "keyword",
                     "index": true
                 }
             }
         }
     }
}`

		// indice doest not exist, let's create it
		req := esapi.IndicesCreateRequest{
			Index: indexName,
			Body:  strings.NewReader(requestBodyData),
		}
		res, err := req.Do(context.Background(), esClient)
		debugESResponse("IndicesCreateRequest:"+indexName, res, err)
		defer res.Body.Close()
		log.Printf("Status Code for IndicesCreateRequest (%s) : %d", indexName, res.StatusCode)
		if res.IsError() {
			var rsp_IndicesCreateRequest map[string]interface{}
			json.NewDecoder(res.Body).Decode(&rsp_IndicesCreateRequest)
			log.Printf("Response for IndicesCreateRequest (%s) : %+v", indexName, rsp_IndicesCreateRequest)
		}

	}

}

// Just to display index settings at startup.
func debugIndexSetting(esClient *elasticsearch6.Client, indexName string) {
	log.Debugf("Get settings for index <%s>", indexName)
	req_settings := esapi.IndicesGetSettingsRequest{
		Index:  []string{indexName},
		Pretty: true,
	}
	res, err := req_settings.Do(context.Background(), esClient)
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

// Perform a refresh query on ES cluster for this particular index.
func (s *elasticStore) refreshIndex(indexName string) {
	req_get := esapi.IndicesRefreshRequest{
		Index:           []string{indexName},
		ExpandWildcards: "none",
		AllowNoIndices:  &pfalse,
	}
	res, err := req_get.Do(context.Background(), s.esClient)
	defer res.Body.Close()
	debugESResponse("IndicesRefreshRequest:"+indexName, res, err)
}

// The document is enriched by adding 'clusterId' and 'iid' properties.
// TODO: Can be optimized by directly adding the entries in byte array (rather than marshaling / unmarshaling twice)
func (s *elasticStore) buildElasticDocument(k string, v interface{}) (string, []byte, error) {
	// Extract indice name and timestamp by parsing the key
	storeType, timestamp := extractStoreTypeAndTimestamp(k)
	log.Debugf("storeType is: %s, timestamp: %s", storeType, timestamp)

	// Convert timestamp to an int64
	eventDate, err := time.Parse(time.RFC3339Nano, timestamp)
	if err != nil {
		return storeType, nil, errors.Wrapf(err, "failed to parse timestamp %+v as time, error was: %+v", timestamp, err)
	}
	// Convert to UnixNano int64
	iid := eventDate.UnixNano()

	// This is the piece of 'JSON' we want to append
	a := `,"iid":"` + strconv.FormatInt(iid, 10) + `","clusterId":"` + s.clusterId + `"`
	// v is a json.RawMessage
	raw := v.(json.RawMessage)
	raw = appendJsonInBytes(raw, []byte(a))

	return storeType, raw, nil
}

// We need to append JSON directly into []byte to avoid useless and costly  marshaling / unmarshaling.
func appendJsonInBytes(a []byte, v []byte) []byte {
	last := len(a) - 1
	lastByte := a[last]
	// just append v at the end
	a = append(a, v...)
	// then slice
	for i, s := range v {
		a[last+i] = s
	}
	a[last+len(v)] = lastByte
	return a
}

// Set index a document (log or event) into ES.
func (s *elasticStore) Set(ctx context.Context, k string, v interface{}) error {
	log.Debugf("Set called will key %s", k)

	if err := utils.CheckKeyAndValue(k, v); err != nil {
		return err
	}

	storeType, body, err := s.buildElasticDocument(k, v)
	if err != nil {
		return err
	}

	indexName := s.getIndexName(storeType)
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

// The index name are prefixed to avoid index name collisions.
func (s *elasticStore) getIndexName(storeType string) string {
	return s.cfg.indicePrefix + storeType
}

// SetCollection index collections using ES bulk requests.
// We consider both 'max_bulk_size' and 'max_bulk_count' to define bulk requests size.
func (s *elasticStore) SetCollection(ctx context.Context, keyValues []store.KeyValueIn) error {
	totalDocumentCount := len(keyValues)
	log.Printf("SetCollection called with an array of size %d", totalDocumentCount)

	if keyValues == nil || totalDocumentCount == 0 {
		return nil
	}

	iterationCount := int(math.Ceil((float64(totalDocumentCount) / float64(s.cfg.maxBulkCount))))
	log.Printf("max_bulk_count is %d, so a minimum of %d iterations will be necessary to bulk index the %d documents", s.cfg.maxBulkCount, iterationCount, totalDocumentCount)

	// The current index in []keyValues (also the number of documents indexed)
	var kvi int = 0
	// The number of iterations
	var i int = 0
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

			kv := keyValues[kvi]
			if err := utils.CheckKeyAndValue(kv.Key, kv.Value); err != nil {
				return err
			}

			storeType, document, err := s.buildElasticDocument(kv.Key, kv.Value)
			if err != nil {
				return err
			}
			log.Debugf("About to add a document of size %d bytes to bulk request", len(document))

			// The bulk action
			index := `{"index":{"_index":"` + s.getIndexName(storeType) + `","_type":"logs_or_event"}}`
			// 2 = len("\n\n")
			// FIXME: not able to make it work defining the slice length
			// 2020/06/09 02:54:20 [FATAL] failed to migrate data from Consul for root path:"_yorc/logs" in store with name:"elastic": Error while sending bulk request, response code was <500> and response message was <[500 Internal Server Error] {"error":{"root_cause":[{"type":"char_conversion_exception","reason":"Invalid UTF-32 character 0x7b21696e (above 0x0010ffff) at char #84, byte #339)"}],"type":"char_conversion_exception","reason":"Invalid UTF-32 character 0x7b21696e (above 0x0010ffff) at char #84, byte #339)"},"status":500}>
			//bulkOperation := make([]byte, len(index) + len(document) + 2)
			bulkOperation := make([]byte, 0)
			bulkOperation = append(bulkOperation, index...)
			bulkOperation = append(bulkOperation, "\n"...)
			bulkOperation = append(bulkOperation, document...)
			bulkOperation = append(bulkOperation, "\n"...)
			log.Debugf("About to add a bulk operation of size %d bytes to bulk request, current size of bulk request body is %d bytes", len(bulkOperation), len(body))

			// 1 = len("\n") the last newline that will be appended to terminate the bulk request
			estimatedBodySize := len(body) + len(bulkOperation) + 1
			if len(bulkOperation) + 1 > maxBulkSizeInBytes {
				return errors.Errorf("A bulk operation size (order + document %s) is greater than the maximum bulk size authorized (%dkB) : %d > %d, this document can't be sent to ES, please adapt your configuration !", kv.Key, s.cfg.maxBulkSize, len(bulkOperation) + 1, maxBulkSizeInBytes)
			}
			if estimatedBodySize > maxBulkSizeInBytes {
				log.Printf("The limit of bulk size (%d kB) will be reached (%d > %d), the current document will be sent in the next bulk request", s.cfg.maxBulkSize, estimatedBodySize, maxBulkSizeInBytes)
				break
			} else {
				log.Debugf("Append document built from key %s to bulk request body, storeType was %s", kv.Key, storeType)

				// Append the bulk operation
				body = append(body, bulkOperation...)

				kvi++;
				bulkActionCount++;
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
		//debugESResponse("BulkRequest", res, err)

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
			} else {
				log.Printf("Bulk request containing %d documents (%d bytes) has been accepted without errors", bulkActionCount, len(body))
			}
		}
		// increment the number of iterations
		i++
	}
	log.Printf("A total of %d documents have been successfully indexed using %d bulk requests", kvi, i)

	return nil
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

// Delete removes ES documents using a deleteByRequest query.
func (s *elasticStore) Delete(ctx context.Context, k string, recursive bool) error {
	log.Debugf("Delete called k: %s, recursive: %t", k, recursive)

	// Extract indice name and deploymentId by parsing the key
	storeType, deploymentId := extractStoreTypeAndDeploymentId(k)
	indexName := s.getIndexName(storeType)
	log.Debugf("storeType is: %s, indexName is %s, deploymentId is: %s", storeType, indexName, deploymentId)

	query := `{"query" : { "bool" : { "must" : [{ "term": { "clusterId" : "` + s.clusterId + `" }}, { "term": { "deploymentId" : "` + deploymentId + `" }}]}}}`
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

func (s *elasticStore) GetLastModifyIndex(k string) (uint64, error) {
	log.Debugf("GetLastModifyIndex called k: %s", k)

	// Extract indice name and deploymentId by parsing the key
	indexName, deploymentId := extractStoreTypeAndDeploymentId(k)
	log.Debugf("indexName is: %s, deploymentId is: %s", indexName, deploymentId)

	return s.internalGetLastModifyIndex(s.getIndexName(indexName), deploymentId)

}

func parseInt64StringToUint64(value string) (uint64, error) {
	valueInt, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, errors.Wrapf(err, "Not able to parse string: %s to int64, error was: %+v", value, err)
	}
	result := uint64(valueInt)
	return result, nil
}

// The last index is found by querying ES using aggregation and a 0 size request.
func (s *elasticStore) internalGetLastModifyIndex(indexName string, deploymentId string) (uint64, error) {

	// The last_index is query by using ES aggregation query ~= MAX(iid) HAVING deploymentId AND clusterId
	query := buildLastModifiedIndexQuery(s.clusterId, deploymentId)
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
	var last_index uint64 = 0
	if hits > 0 {
		last_index, err = parseInt64StringToUint64(r.aggregations.logs_or_events.last_index.value)
		if err != nil {
			return last_index, err
		}
	}

	// Print the response status, number of results, and request duration.
	log.Debugf(
		"[%s] %d hits; last_index: %d",
		res.Status(),
		hits,
		last_index,
	)

	return last_index, nil
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
	storeType, deploymentId := extractStoreTypeAndDeploymentId(k)
	indexName := s.getIndexName(storeType)
	log.Debugf("storeType is: %s, indexName is: %s, deploymentId is: %s", storeType, indexName, deploymentId)

	query := getListQuery(s.clusterId, deploymentId, waitIndex, 0)

	now := time.Now()
	end := now.Add(timeout - s.cfg.esRefreshWaitTimeout)
	log.Debugf("Now is : %v, date after timeout will be %v (ES timeout duration will be %v)", now, end, timeout-s.cfg.esRefreshWaitTimeout)
	var values = make([]store.KeyValueOut, 0)
	var lastIndex = waitIndex
	var hits = 0
	var err error
	for {
		// first just query to know if they is something to fetch, we just want the max iid (so order desc, size 1)
		hits, values, lastIndex, err = s.doQueryEs(indexName, query, waitIndex, 1, "desc");
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
		query := getListQuery(s.clusterId, deploymentId, waitIndex, lastIndex)
		if s.cfg.esForceRefresh {
			// force refresh for this index
			s.refreshIndex(indexName);
		}
		time.Sleep(s.cfg.esRefreshWaitTimeout)
		oldHits := hits
		hits, values, lastIndex, err = s.doQueryEs(indexName, query, waitIndex, 10000, "asc");
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

// This ES aggregation query is built using clusterId and eventually deploymentId.
func buildLastModifiedIndexQuery(clusterId string, deploymentId string) string {
	var query string
	if len(deploymentId) == 0 {
		query = `
{
    "aggs" : {
        "logs_or_events" : {
            "filter" : {
				"term": { "clusterId": "` + clusterId + `" }
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
        "logs_or_events" : {
            "filter" : {
                "bool": {
                    "must": [
                        { "term": { "deploymentId": "` + deploymentId + `" } },
                        { "term": { "clusterId": "` + clusterId + `" } }
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
	return query
}

// The max uint is 9223372036854775807 (19 cars), this is the maximum nano for a time (2262-04-12 00:47:16.854775807 +0100 CET).
// Since we use nanotimestamp as ID for events and logs, and since we store this ID as string in ES index, we must ensure that
// the string will be comparable.
func getSortableStringFromUint64(nanoTimestamp uint64) string {
	nanoTimestampStr := strconv.FormatUint(nanoTimestamp, 10)
	if len(nanoTimestampStr) < 19 {
		nanoTimestampStr = strings.Repeat("0", 19-len(nanoTimestampStr)) + nanoTimestampStr
	}
	return nanoTimestampStr
}

// This ES range query is built using 'waitIndex' and eventually 'maxIndex' and filtered using 'clusterId' and eventually 'deploymentId'.
func getListQuery(clusterId string, deploymentId string, waitIndex uint64, maxIndex uint64) string {

	var rangeQuery, query string
	if maxIndex > 0 {
		rangeQuery = `
            {
               "range":{
                  "iid":{
                     "gt": "` + getSortableStringFromUint64(waitIndex) + `",
					 "lte": "` + getSortableStringFromUint64(maxIndex) + `"
                  }
               }
            }`
	} else {
		rangeQuery = `
            {
               "range":{
                  "iid":{
                     "gt": "` + getSortableStringFromUint64(waitIndex) + `"
                  }
               }
            }`
	}
	if len(deploymentId) == 0 {
		query = `
{
   "query":{
      "bool":{
         "must":[
            {
               "term":{
                  "clusterId":"` + clusterId + `"
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
                  "clusterId":"` + clusterId + `"
               }
            },
            {
               "term":{
                  "deploymentId":"` + deploymentId + `"
               }
            },` + rangeQuery + `
         ]
      }
   }
}`
	}
	return query
}

// Query ES for events or logs specifying the expected results 'size' and the sort 'order'.
func (s *elasticStore) doQueryEs(index string, query string, waitIndex uint64, size int, order string) (int, []store.KeyValueOut, uint64, error) {
	log.Debugf("Search ES %s using query: %s", index, query)

	values := make([]store.KeyValueOut, 0)

	res, err := s.esClient.Search(
		s.esClient.Search.WithContext(context.Background()),
		s.esClient.Search.WithIndex(index),
		s.esClient.Search.WithSize(size),
		s.esClient.Search.WithBody(strings.NewReader(query)),
		// important sort on iid
		s.esClient.Search.WithSort("iid:"+order),
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
		} else {
			log.Printf("An error occurred while performing ES search on index %s, query was: <%s>, response code was %d (%s). Response body was: %+v", index, query, res.StatusCode, res.Status(), e)
			return 0, values, waitIndex, errors.Wrapf(err, "An error occurred while performing ES search on index %s, query was: <%s>, response code was %d (%s). Response body was: %+v", index, query, res.StatusCode, res.Status(), e)
		}
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
		iid_int64, err := parseInt64StringToUint64(iid.(string))
		if err != nil {
			log.Printf("Not able to parse iid_str property %s as uint64, document id: %s, source: %+v, ignoring this document !", iid, id, source);
		} else {
			jsonString, err := json.Marshal(source)
			if err != nil {
				log.Printf("Not able to marshall document source, document id: %s, source: %+v, ignoring this document !", id, source);
			} else {
				// since the result is sorted on iid, we can use the last hit to define lastIndex
				lastIndex = iid_int64
				// append value to result
				values = append(values, store.KeyValueOut{
					Key:             id,
					LastModifyIndex: iid_int64,
					Value:           source,
					RawValue:        jsonString,
				})
			}
		}
	}

	log.Debugf("doQueryEs called result waitIndex: %d, LastIndex: %d, len(values): %d", waitIndex, lastIndex, len(values))
	return hits, values, lastIndex, nil
}
