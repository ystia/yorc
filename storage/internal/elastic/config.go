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
	"github.com/spf13/cast"
	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/log"
	"reflect"
	"time"
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

// Get the tag for this field (for internal usage only: fatal if not found !).
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
	k = getElasticStorageConfigPropertyTag("esForceRefresh", "json")
	if storeProperties.IsSet(k) {
		cfg.esForceRefresh = storeProperties.GetBool(k)
	} else {
		cfg.esForceRefresh = cast.ToBool(getElasticStorageConfigPropertyTag("esForceRefresh", "default"))
	}
	k = getElasticStorageConfigPropertyTag("indicePrefix", "json")
	if storeProperties.IsSet(k) {
		cfg.indicePrefix = storeProperties.GetString(k)
	} else {
		cfg.indicePrefix = getElasticStorageConfigPropertyTag("indicePrefix", "default")
	}
	cfg.esQueryPeriod = getDurationFromSettingsOrDefaults("esQueryPeriod", storeProperties)
	cfg.esRefreshWaitTimeout = getDurationFromSettingsOrDefaults("esRefreshWaitTimeout", storeProperties)
	cfg.maxBulkSize = getIntFromSettingsOrDefaults("maxBulkSize", storeProperties)
	cfg.maxBulkCount = getIntFromSettingsOrDefaults("maxBulkCount", storeProperties)

	return cfg
}

// Get the duration from store config properties, fallback to required default value defined in struc.
func getDurationFromSettingsOrDefaults(fn string, dm config.DynamicMap) time.Duration {
	k := getElasticStorageConfigPropertyTag(fn, "json")
	if dm.IsSet(k) {
		return dm.GetDuration(k)
	}
	return cast.ToDuration(getElasticStorageConfigPropertyTag(fn, "default"))
}

// Get the int from store config properties, fallback to required default value defined in struc.
func getIntFromSettingsOrDefaults(fn string, dm config.DynamicMap) int {
	k := getElasticStorageConfigPropertyTag(fn, "json")
	if dm.IsSet(k) {
		return dm.GetInt(k)
	}
	return cast.ToInt(getElasticStorageConfigPropertyTag(fn, "default"))
}
