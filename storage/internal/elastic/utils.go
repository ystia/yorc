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
	"encoding/json"
	"github.com/pkg/errors"
	"github.com/ystia/yorc/v4/log"
	"github.com/ystia/yorc/v4/storage/store"
	"github.com/ystia/yorc/v4/storage/utils"
	"regexp"
	"strconv"
	"strings"
	"time"
)

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
var storeTypeAndDeploymentIDRegex = regexp.MustCompile(`(?m)\_yorc\/(\w+)\/?(.+)?\/?`)

// Parse a key of form "_yorc/events/" or "_yorc/logs/MyApp" or "_yorc/logs/MyApp/" to get the store type (logs|events) and eventually the deploymentId.
func extractStoreTypeAndDeploymentID(k string) (storeType string, deploymentID string) {
	res := storeTypeAndDeploymentIDRegex.FindAllStringSubmatch(k, -1)
	for i := range res {
		storeType = res[i][1]
		if len(res[i]) == 3 {
			deploymentID = res[i][2]
			if strings.HasSuffix(deploymentID, "/") {
				deploymentID = deploymentID[:len(deploymentID)-1]
			}
		}
	}
	return storeType, deploymentID
}

// We need to append JSON directly into []byte to avoid useless and costly marshaling / unmarshaling.
func appendJSONInBytes(a []byte, v []byte) []byte {
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

func parseInt64StringToUint64(value string) (uint64, error) {
	valueInt, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, errors.Wrapf(err, "Not able to parse string: %s to int64, error was: %+v", value, err)
	}
	result := uint64(valueInt)
	return result, nil
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

// The document is enriched by adding 'clusterId' and 'iid' properties.
// This addition is done by directly manipulating the []byte in order to avoid costly successive marshal / unmarshal operations.
func buildElasticDocument(clusterID string, k string, rawMessage interface{}) (string, []byte, error) {
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
	a := `,"iid":"` + strconv.FormatInt(iid, 10) + `","clusterId":"` + clusterID + `"`
	// v is a json.RawMessage
	raw := rawMessage.(json.RawMessage)
	raw = appendJSONInBytes(raw, []byte(a))

	return storeType, raw, nil
}

// An error is returned if :
// - it's not valid (key or value nil)
// - the size of the resulting bulk operation exceed the maximum authorized for a bulk request
// The value is not added if it's size + the current body size exceed the maximum authorized for a bulk request.
// Return a bool indicating if the value has been added to the bulk request body.
func eventuallyAppendValueToBulkRequest(c elasticStoreConf, clusterID string, body *[]byte, kv store.KeyValueIn, maxBulkSizeInBytes int) (bool, error) {
	if err := utils.CheckKeyAndValue(kv.Key, kv.Value); err != nil {
		return false, err
	}

	storeType, document, err := buildElasticDocument(clusterID, kv.Key, kv.Value)
	if err != nil {
		return false, err
	}
	log.Debugf("About to add a document of size %d bytes to bulk request", len(document))

	// The bulk action
	index := `{"index":{"_index":"` + getIndexName(c, storeType) + `","_type":"logs_or_event"}}`
	bulkOperation := make([]byte, 0)
	bulkOperation = append(bulkOperation, index...)
	bulkOperation = append(bulkOperation, "\n"...)
	bulkOperation = append(bulkOperation, document...)
	bulkOperation = append(bulkOperation, "\n"...)
	log.Debugf("About to add a bulk operation of size %d bytes to bulk request, current size of bulk request body is %d bytes", len(bulkOperation), len(*body))

	// 1 = len("\n") the last newline that will be appended to terminate the bulk request
	estimatedBodySize := len(*body) + len(bulkOperation) + 1
	if len(bulkOperation)+1 > maxBulkSizeInBytes {
		return false, errors.Errorf(
			"A bulk operation size (order + document %s) is greater than the maximum bulk size authorized (%dkB) : %d > %d, this document can't be sent to ES, please adapt your configuration !",
			kv.Key, c.maxBulkSize, len(bulkOperation)+1, maxBulkSizeInBytes,
		)
	}
	if estimatedBodySize > maxBulkSizeInBytes {
		log.Printf(
			"The limit of bulk size (%d kB) will be reached (%d > %d), the current document will be sent in the next bulk request",
			c.maxBulkSize, estimatedBodySize, maxBulkSizeInBytes,
		)
		return false, nil
	}
	log.Debugf("Append document built from key %s to bulk request body, storeType was %s", kv.Key, storeType)
	// Append the bulk operation
	*body = append(*body, bulkOperation...)
	return true, nil
}

// The index name are prefixed to avoid index name collisions.
func getIndexName(c elasticStoreConf, storeType string) string {
	return c.indicePrefix + storeType
}
