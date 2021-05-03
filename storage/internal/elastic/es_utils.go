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
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	elasticsearch6 "github.com/elastic/go-elasticsearch/v6"
	"github.com/elastic/go-elasticsearch/v6/esapi"
	"github.com/pkg/errors"
	"github.com/ystia/yorc/v4/log"
	"github.com/ystia/yorc/v4/storage/store"
)

var pfalse = false

func prepareEsClient(elasticStoreConfig elasticStoreConf) (*elasticsearch6.Client, error) {
	log.Printf("Elastic storage will run using this configuration: %+v", elasticStoreConfig)

	esConfig := elasticsearch6.Config{Addresses: elasticStoreConfig.esUrls}

	if len(elasticStoreConfig.caCertPath) > 0 {
		log.Printf("Reading CACert file from %s", elasticStoreConfig.caCertPath)
		caCert, err := ioutil.ReadFile(elasticStoreConfig.caCertPath)
		if err != nil {
			return nil, errors.Wrapf(err, "Not able to read Cert file from <%s>", elasticStoreConfig.caCertPath)
		}
		esConfig.CACert = caCert

		if len(elasticStoreConfig.certPath) > 0 && len(elasticStoreConfig.keyPath) > 0 {
			cert, err := tls.LoadX509KeyPair(elasticStoreConfig.certPath, elasticStoreConfig.keyPath)
			if err != nil {
				return nil, errors.Wrapf(err, "Not able to read cert and/or key file from <%s> and <%s>", elasticStoreConfig.certPath, elasticStoreConfig.keyPath)
			}
			caCertPool := x509.NewCertPool()
			caCertPool.AppendCertsFromPEM(caCert)

			// Setup HTTPS client
			tlsConfig := &tls.Config{
				Certificates: []tls.Certificate{cert},
				RootCAs:      caCertPool,
			}
			tlsConfig.BuildNameToCertificate()
			transport := &http.Transport{TLSClientConfig: tlsConfig}

			esConfig.Transport = transport
		}
	}
	if log.IsDebug() || elasticStoreConfig.traceRequests {
		// In debug mode or when traceRequests option is activated, we add a custom logger that print requests & responses
		log.Printf("\t- Tracing ES requests & response can be expensive and verbose !")
		esConfig.Logger = &debugLogger{}
	} else {
		// otherwise log only failure are logger
		esConfig.Logger = &defaultLogger{}
	}

	log.Printf("\t- Index prefix will be %s", elasticStoreConfig.indicePrefix+elasticStoreConfig.clusterID+"_")
	log.Printf("\t- Will query ES for logs or events every %v and will wait for index refresh during %v",
		elasticStoreConfig.esQueryPeriod, elasticStoreConfig.esRefreshWaitTimeout)
	willRefresh := ""
	if !elasticStoreConfig.esForceRefresh {
		willRefresh = "not "
	}
	log.Printf("\t- Will %srefresh index before waiting for indexation", willRefresh)
	log.Printf("\t- While migrating data, the max bulk request size will be %d documents and will never exceed %d kB",
		elasticStoreConfig.maxBulkCount, elasticStoreConfig.maxBulkSize)
	if log.IsDebug() {
		log.Printf("\t- Will use this ES client configuration: %+v", esConfig)
	}

	esClient, e := elasticsearch6.NewClient(esConfig)
	if e != nil {
		return nil, errors.Wrapf(e, "Not able build ES client")
	}
	infoResponse, e := esClient.Info()
	if e != nil {
		return nil, errors.Wrapf(e, "The ES cluster info request failed")
	}
	log.Printf("Here is the ES cluster info: %+v", infoResponse)
	return esClient, e
}

// Init ES index for logs or events storage: create it if not found.
func initStorageIndex(c *elasticsearch6.Client, elasticStoreConfig elasticStoreConf, storeType string) error {

	indexName := getIndexName(elasticStoreConfig, storeType)
	log.Printf("Checking if index <%s> already exists", indexName)

	// check if the sequences index exists
	req := esapi.IndicesExistsRequest{
		Index:           []string{indexName},
		ExpandWildcards: "none",
		AllowNoIndices:  &pfalse,
	}
	res, err := req.Do(context.Background(), c)
	defer closeResponseBody("IndicesExistsRequest:"+indexName, res)

	if err != nil {
		return err
	}

	if res.StatusCode == 200 {
		log.Printf("Indice %s was found, nothing to do !", indexName)
		return nil
	} else if res.StatusCode == 404 {
		log.Printf("Indice %s was not found, let's create it !", indexName)

		requestBodyData := buildInitStorageIndexQuery(elasticStoreConfig)

		// indice doest not exist, let's create it
		req := esapi.IndicesCreateRequest{
			Index: indexName,
			Body:  strings.NewReader(requestBodyData),
		}
		res, err := req.Do(context.Background(), c)
		defer closeResponseBody("IndicesCreateRequest:"+indexName, res)
		if err = handleESResponseError(res, "IndicesCreateRequest:"+indexName, requestBodyData, err); err != nil {
			return err
		}
	} else {
		return handleESResponseError(res, "IndicesExistsRequest:"+indexName, "", err)
	}
	return nil
}

// Perform a refresh query on ES cluster for this particular index.
func refreshIndex(c *elasticsearch6.Client, indexName string) {
	req := esapi.IndicesRefreshRequest{
		Index:           []string{indexName},
		ExpandWildcards: "none",
		AllowNoIndices:  &pfalse,
	}
	res, err := req.Do(context.Background(), c)
	err = handleESResponseError(res, "IndicesRefreshRequest:"+indexName, "", err)
	if err != nil {
		log.Println("An error occurred while refreshing index, due to : %+v", err)
	}
	defer closeResponseBody("IndicesRefreshRequest:"+indexName, res)
}

// Query ES for events or logs specifying the expected results 'size' and the sort 'order'.
func doQueryEs(ctx context.Context, c *elasticsearch6.Client, conf elasticStoreConf,
	index string,
	query string,
	waitIndex uint64,
	size int,
	order string,
) (hits int, values []store.KeyValueOut, lastIndex uint64, err error) {

	log.Debugf("Search ES %s using query: %s", index, query)
	lastIndex = waitIndex

	res, e := c.Search(
		c.Search.WithContext(ctx),
		c.Search.WithIndex(index),
		c.Search.WithSize(size),
		c.Search.WithBody(strings.NewReader(query)),
		// important sort on iid
		c.Search.WithSort("iid:"+order),
	)
	if e != nil {
		err = errors.Wrapf(e, "Failed to perform ES search on index %s, query was: <%s>, error was: %+v", index, query, e)
		return
	}
	defer closeResponseBody("Search:"+index, res)

	err = handleESResponseError(res, "Search:"+index, query, e)
	if err != nil {
		return
	}

	var r map[string]interface{}
	if decodeErr := json.NewDecoder(res.Body).Decode(&r); decodeErr != nil {
		err = errors.Wrapf(decodeErr,
			"Not able to decode ES response while performing ES search on index %s, query was: <%s>, response code was %d (%s)",
			index, query, res.StatusCode, res.Status(),
		)
		return
	}

	logShardsInfos(r)

	hits = int(r["hits"].(map[string]interface{})["total"].(float64))
	duration := int(r["took"].(float64))
	log.Debugf("Search ES request on index %s took %dms, hits=%d, response code was %d (%s)", index, duration, hits, res.StatusCode, res.Status())

	lastIndex = decodeEsQueryResponse(conf, index, waitIndex, size, r, &values)

	log.Debugf("doQueryEs called result waitIndex: %d, LastIndex: %d, len(values): %d", waitIndex, lastIndex, len(values))
	return hits, values, lastIndex, nil
}

// Decode the response and define the last index
func decodeEsQueryResponse(conf elasticStoreConf, index string, waitIndex uint64, size int, r map[string]interface{}, values *[]store.KeyValueOut) (lastIndex uint64) {
	lastIndex = waitIndex
	// Print the ID and document source for each hit.
	i := 0
	for _, hit := range r["hits"].(map[string]interface{})["hits"].([]interface{}) {
		id := hit.(map[string]interface{})["_id"].(string)
		source := hit.(map[string]interface{})["_source"].(map[string]interface{})
		iid := source["iidStr"]
		iidUInt64, err := parseInt64StringToUint64(iid.(string))
		if err != nil {
			log.Printf("Not able to parse iid_str property %s as uint64, document id: %s, source: %+v, ignoring this document !", iid, id, source)
		} else {
			jsonString, err := json.Marshal(source)
			if err != nil {
				log.Printf("Not able to marshall document source, document id: %s, source: %+v, ignoring this document !", id, source)
			} else {
				// since the result is sorted on iid, we can use the last hit to define lastIndex
				lastIndex = iidUInt64
				if conf.traceEvents {
					i++
					waitTimestamp := _getTimestampFromUint64(waitIndex)
					iidInt64 := _parseInt64StringToInt64(iid.(string))
					iidTimestamp := time.Unix(0, iidInt64)
					log.Printf("ESList-%s;%d,%v,%d,%d,%s,%v,%d,%d",
						index, waitIndex, waitTimestamp, size, i, iid, iidTimestamp, iidInt64, lastIndex)
				}
				// append value to result
				*values = append(*values, store.KeyValueOut{
					Key:             id,
					LastModifyIndex: iidUInt64,
					Value:           source,
					RawValue:        jsonString,
				})
			}
		}
	}
	return
}

// Send the bulk request to ES and ensure no error is returned.
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
	defer closeResponseBody("BulkRequest", res)

	if err != nil {
		return err
	} else if res.IsError() {
		return handleESResponseError(res, "BulkRequest", string(*body), err)
	} else {
		var rsp map[string]interface{}
		err = json.NewDecoder(res.Body).Decode(&rsp)
		if err != nil {
			// Don't know if the bulk request response contains error so fail by default
			return errors.Errorf(
				"The bulk request succeeded (%s), but not able to decode the response, so not able to determine if bulk operations are correctly handled",
				res.Status(),
			)
		}
		if rsp["errors"].(bool) {
			// The bulk request contains errors
			return errors.Errorf("The bulk request succeeded, but the response contains errors : %+v", rsp)
		}
	}
	log.Printf("Bulk request containing %d operations (%d bytes) has been accepted successfully", opeCount, len(*body))
	return nil
}

// Consider the ES Response and wrap errors when needed
func handleESResponseError(res *esapi.Response, requestDescription string, query string, requestError error) error {
	if requestError != nil {
		return errors.Wrapf(requestError, "Error while sending %s, query was: %s", requestDescription, query)
	}
	if res.IsError() {
		return errors.Errorf(
			"An error was returned by ES while sending %s, status was %s, query was: %s, response: %+v",
			requestDescription, res.Status(), query, res.String())
	}
	return nil
}

// Close response body, if an error occur, just print it
func closeResponseBody(requestDescription string, res *esapi.Response) {
	if res != nil && res.Body != nil {
		err := res.Body.Close()
		if err != nil {
			log.Printf("[%s] Was not able to close resource response body, error was: %+v", requestDescription, err)
		}
	}
}

// Log shards stats
func logShardsInfos(r map[string]interface{}) {
	si := r["_shards"].(map[string]interface{})

	duration := int(r["took"].(float64))

	tt := int(si["total"].(float64))
	ts := int(si["successful"].(float64))

	if ts < tt {
		log.Printf("[Warn] ES Uncomplete response: %d/%d shards (%dms)", ts, tt, duration)
	}
}

type debugLogger struct{}

// RequestBodyEnabled makes the client pass request body to logger
func (l *debugLogger) RequestBodyEnabled() bool { return true }

// ResponseBodyEnabled makes the client pass response body to logger
func (l *debugLogger) ResponseBodyEnabled() bool { return true }

// LogRoundTrip will use log to debug ES request and response (when debug is activated)
func (l *debugLogger) LogRoundTrip(
	req *http.Request,
	res *http.Response,
	err error,
	start time.Time,
	dur time.Duration,
) error {

	var level string
	switch {
	case err != nil:
		level = "Exception"
	case res != nil && res.StatusCode > 0 && res.StatusCode < 300:
		level = "Success"
	case res != nil && res.StatusCode > 299 && res.StatusCode < 500:
		level = "Warn"
	case res != nil && res.StatusCode > 499:
		level = "Error"
	default:
		level = "Unknown"
	}

	var reqBuffer, resBuffer bytes.Buffer
	if req != nil && req.Body != nil && req.Body != http.NoBody {
		// We explicitly ignore errors here since it's a debug feature
		_, _ = io.Copy(&reqBuffer, req.Body)
	}
	reqStr := reqBuffer.String()
	if res != nil && res.Body != nil && res.Body != http.NoBody {
		// We explicitly ignore errors here since it's a debug feature
		_, _ = io.Copy(&resBuffer, res.Body)
	}
	resStr := resBuffer.String()
	log.Printf("ES Request [%s][%v][%s][%s][%d][%v] [%+v] : [%+v]",
		level, start, req.Method, req.URL.String(), res.StatusCode, dur, reqStr, resStr)

	return nil
}

type defaultLogger struct{}

// RequestBodyEnabled makes the client pass request body to logger
func (l *defaultLogger) RequestBodyEnabled() bool { return true }

// ResponseBodyEnabled makes the client pass response body to logger
func (l *defaultLogger) ResponseBodyEnabled() bool { return true }

// LogRoundTrip will use log to debug ES request and response (when debug is activated)
func (l *defaultLogger) LogRoundTrip(
	req *http.Request,
	res *http.Response,
	err error,
	start time.Time,
	dur time.Duration,
) error {

	var level string
	var errType string
	var errReason string

	switch {
	case err != nil:
		level = "Exception"
	case res != nil && res.StatusCode > 0 && res.StatusCode < 300:
		return nil
	case res != nil && res.StatusCode > 299 && res.StatusCode < 500:
		level = "Warn"
	case res != nil && res.StatusCode > 499:
		errType, errReason = extractEsError(res)
		level = "Error"
	default:
		level = "Unknown"
	}

	if errType == "" && errReason == "" {
		log.Printf("ES Request [%s][%v][%s][%s][%d][%v]",
			level, start, req.Method, req.URL.String(), res.StatusCode, dur)
	} else {
		log.Printf("ES Request [%s][%v][%s][%s][%d][%v][%s][%s]",
			level, start, req.Method, req.URL.String(), res.StatusCode, dur, errType, errReason)
	}
	return nil
}

func extractEsError(response *http.Response) (errType, errReason string) {
	var rb map[string]interface{}
	var rv interface{}
	var ok bool

	errType = "N/A"
	errReason = "N/A"

	if err := json.NewDecoder(response.Body).Decode(&rb); err != nil {
		return
	}

	if rv, ok = rb["error"]; !ok {
		return
	}

	if rb, ok = rv.(map[string]interface{}); !ok {
		return
	}

	if rv, ok = rb["type"]; ok {
		errType, _ = rv.(string)
	}

	if rv, ok = rb["reason"]; ok {
		errReason, _ = rv.(string)
	}

	return
}
