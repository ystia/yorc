package elastic

import (
	"context"
	"encoding/json"
	"github.com/elastic/go-elasticsearch/v6"
	"github.com/elastic/go-elasticsearch/v6/esapi"
	"github.com/pkg/errors"
	"github.com/ystia/yorc/v4/log"
	"regexp"
	"strconv"
	"strings"
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

// Init ES index for logs or events storage: create it if not found.
func initStorageIndex(esClient *v6.Client, indexName string) {

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
		res, err := req.Do(context.Background(), esClient)
		debugESResponse("IndicesCreateRequest:"+indexName, res, err)
		defer res.Body.Close()
		if res.IsError() {
			var rsp map[string]interface{}
			json.NewDecoder(res.Body).Decode(&rsp)
			log.Printf("Response for IndicesCreateRequest (%s) : %+v", indexName, rsp)
		}

	}

}

// Just to display index settings at startup.
func debugIndexSetting(esClient *v6.Client, indexName string) {
	log.Debugf("Get settings for index <%s>", indexName)
	req := esapi.IndicesGetSettingsRequest{
		Index:  []string{indexName},
		Pretty: true,
	}
	res, err := req.Do(context.Background(), esClient)
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

// We need to append JSON directly into []byte to avoid useless and costly  marshaling / unmarshaling.
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

