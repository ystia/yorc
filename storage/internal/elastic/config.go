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
	"reflect"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cast"
	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/log"
)

var elasticStoreConfType = reflect.TypeOf(elasticStoreConf{})

// elasticStoreConf represents the elastic store configuration that can be set in store.properties configuration.
type elasticStoreConf struct {
	// The ES cluster urls (array or CSV)
	esUrls []string `json:"es_urls"`
	// The path to the CACert file when TLS is activated for ES
	caCertPath string `json:"ca_cert_path"`
	// The path to a PEM encoded certificate file when TLS is activated for ES
	certPath string `json:"cert_path"`
	// The path to a PEM encoded private key file when TLS is activated for ES
	keyPath string `json:"key_path"`
	// All index used by yorc will be prefixed by this prefix
	indicePrefix string `json:"index_prefix" default:"yorc_"`
	// When querying logs and event, we wait this timeout before each request when it returns nothing
	// (until something is returned or the waitTimeout is reached)
	esQueryPeriod time.Duration `json:"es_query_period" default:"4s"`
	// This timeout is used to wait for more than refresh_interval = 1s when querying logs and events indexes
	esRefreshWaitTimeout time.Duration `json:"es_refresh_wait_timeout" default:"2s"`
	// When querying ES, force refresh index before waiting for refresh
	esForceRefresh bool `json:"es_force_refresh" default:"false"`
	// This is the maximum size (in kB) of bulk request sent while migrating data
	maxBulkSize int `json:"max_bulk_size" default:"4000"`
	// This is the maximum size (in term of number of documents) of bulk request sent while migrating data
	maxBulkCount int `json:"max_bulk_count" default:"1000"`
	// This optional ID will be used to distinguish logs & events in the indexes. If not set, we'll use the Consul.Datacenter
	clusterID string `json:"cluster_id"`
	// Set to true if you want to print ES requests (for debug only)
	traceRequests bool `json:"trace_requests" default:"false"`
	// Set to true if you want to trace events & logs when sent (for debug only)
	traceEvents bool `json:"trace_events" default:"false"`
	// Inital shards at index creation
	InitialShards int `json:"initial_shards" default:"-1"`
	// Initial replicas at index creation
	InitialReplicas int `json:"initial_replicas" default:"-1"`
}

// Get the tag for this field (for internal usage only: fatal if not found !).
func getElasticStorageConfigPropertyTag(fn string, tn string) (tagValue string, e error) {
	f, found := elasticStoreConfType.FieldByName(fn)
	if !found {
		e = errors.Errorf("Not able to get field %s on elasticStoreConf struct, there is an issue with this code !", fn)
		return
	}
	tagValue = f.Tag.Get(tn)
	if tagValue == "" {
		e = errors.Errorf("Not able to get field %s's tag %s value on elasticStoreConf struct, there is an issue with this code !\"", fn, tn)
		return
	}
	return
}

// The configuration of the elastic store is defined regarding default values and store 'properties' dynamic map.
// Will fail if the required es_urls is not set in store 'properties'
func getElasticStoreConfig(yorcConfig config.Configuration, storeConfig config.Store) (cfg elasticStoreConf, e error) {
	storeProperties := storeConfig.Properties
	var t string
	// The ES urls is required
	t, e = getElasticStorageConfigPropertyTag("esUrls", "json")
	if e != nil {
		return
	}
	if storeProperties.IsSet(t) {
		cfg.esUrls = storeProperties.GetStringSlice(t)
		if cfg.esUrls == nil || len(cfg.esUrls) == 0 {
			e = errors.Errorf("Not able to get ES configuration for elastic store, es_urls store property seems empty : %+v", storeProperties.Get(t))
			return
		}
	} else {
		log.Fatal("Not able to get ES configuration for elastic store, es_urls store property should be set !")
	}
	// Define the clusterID
	t, e = getElasticStorageConfigPropertyTag("clusterID", "json")
	if e != nil {
		return
	}
	if storeProperties.IsSet(t) {
		cfg.clusterID = storeProperties.GetString(t)
	}
	if len(cfg.clusterID) == 0 {
		cfg.clusterID = yorcConfig.Consul.Datacenter
		log.Printf("clusterID not provided or empty, using consul datacenter (%s) as clusterID", yorcConfig.Consul.Datacenter)
	}
	if len(cfg.clusterID) == 0 {
		e = errors.Errorf("Not able to define clusterID, please check configuration !")
		return
	}
	// Define store optional / default configuration
	t, e = getElasticStorageConfigPropertyTag("caCertPath", "json")
	if e != nil {
		return
	}

	if storeProperties.IsSet(t) {
		cfg.caCertPath = storeProperties.GetString(t)
	}
	t, e = getElasticStorageConfigPropertyTag("certPath", "json")
	if e != nil {
		return
	}

	if storeProperties.IsSet(t) {
		cfg.certPath = storeProperties.GetString(t)
	}
	t, e = getElasticStorageConfigPropertyTag("keyPath", "json")
	if e != nil {
		return
	}

	if storeProperties.IsSet(t) {
		cfg.keyPath = storeProperties.GetString(t)
	}
	cfg.esForceRefresh, e = getBoolFromSettingsOrDefaults("esForceRefresh", storeProperties)
	if e != nil {
		return
	}

	t, e = getElasticStorageConfigPropertyTag("indicePrefix", "json")
	if e != nil {
		return
	}

	if storeProperties.IsSet(t) {
		cfg.indicePrefix = storeProperties.GetString(t)
	} else {
		cfg.indicePrefix, e = getElasticStorageConfigPropertyTag("indicePrefix", "default")
		if e != nil {
			return
		}
	}
	cfg.esQueryPeriod, e = getDurationFromSettingsOrDefaults("esQueryPeriod", storeProperties)
	if e != nil {
		return
	}

	cfg.esRefreshWaitTimeout, e = getDurationFromSettingsOrDefaults("esRefreshWaitTimeout", storeProperties)
	if e != nil {
		return
	}

	cfg.maxBulkSize, e = getIntFromSettingsOrDefaults("maxBulkSize", storeProperties)
	if e != nil {
		return
	}
	cfg.maxBulkCount, e = getIntFromSettingsOrDefaults("maxBulkCount", storeProperties)
	if e != nil {
		return
	}

	cfg.traceRequests, e = getBoolFromSettingsOrDefaults("traceRequests", storeProperties)
	if e != nil {
		return
	}

	cfg.traceEvents, e = getBoolFromSettingsOrDefaults("traceEvents", storeProperties)
	if e != nil {
		return
	}

	cfg.InitialShards, e = getIntFromSettingsOrDefaults("InitialShards", storeProperties)
	if e != nil {
		return
	}

	cfg.InitialReplicas, e = getIntFromSettingsOrDefaults("InitialReplicas", storeProperties)
	if e != nil {
		return
	}

	return
}

// Get the duration from store config properties, fallback to required default value defined in struc.
func getDurationFromSettingsOrDefaults(fn string, dm config.DynamicMap) (v time.Duration, er error) {
	t, er := getElasticStorageConfigPropertyTag(fn, "json")
	if er != nil {
		return
	}
	if dm.IsSet(t) {
		v = dm.GetDuration(t)
		return
	}
	t, er = getElasticStorageConfigPropertyTag(fn, "default")
	if er != nil {
		return
	}
	v = cast.ToDuration(t)
	return
}

// Get the int from store config properties, fallback to required default value defined in struc.
func getIntFromSettingsOrDefaults(fn string, dm config.DynamicMap) (v int, err error) {
	t, err := getElasticStorageConfigPropertyTag(fn, "json")
	if err != nil {
		return
	}
	if dm.IsSet(t) {
		v = dm.GetInt(t)
		return
	}
	t, err = getElasticStorageConfigPropertyTag(fn, "default")
	if err != nil {
		return
	}
	v = cast.ToInt(t)
	return
}

// Get the bool from store config properties, fallback to required default value defined in struc.
func getBoolFromSettingsOrDefaults(fn string, dm config.DynamicMap) (v bool, e error) {
	t, e := getElasticStorageConfigPropertyTag(fn, "json")
	if e != nil {
		return
	}
	if dm.IsSet(t) {
		v = dm.GetBool(t)
		return
	}
	t, e = getElasticStorageConfigPropertyTag(fn, "default")
	if e != nil {
		return
	}
	v = cast.ToBool(t)
	return
}
