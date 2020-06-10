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
	"github.com/pkg/errors"
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
func getElasticStorageConfigPropertyTag(fn string, tn string) (tagValue string, e error) {
	t := reflect.TypeOf(elasticStoreConf{})
	f, found := t.FieldByName(fn)
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
func getElasticStoreConfig(storeConfig config.Store) (cfg elasticStoreConf, e error) {
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

	// Define store optional / default configuration
	t, e = getElasticStorageConfigPropertyTag("caCertPath", "json")
	if storeProperties.IsSet(t) {
		cfg.caCertPath = storeProperties.GetString(t)
	}
	t, e = getElasticStorageConfigPropertyTag("esForceRefresh", "json")
	if storeProperties.IsSet(t) {
		cfg.esForceRefresh = storeProperties.GetBool(t)
	} else {
		cfg.esForceRefresh, e = getBoolFromSettingsOrDefaults("esForceRefresh", storeProperties)
	}
	t, e = getElasticStorageConfigPropertyTag("indicePrefix", "json")
	if storeProperties.IsSet(t) {
		cfg.indicePrefix = storeProperties.GetString(t)
	} else {
		cfg.indicePrefix, e = getElasticStorageConfigPropertyTag("indicePrefix", "default")
	}
	cfg.esQueryPeriod, e = getDurationFromSettingsOrDefaults("esQueryPeriod", storeProperties)
	cfg.esRefreshWaitTimeout, e = getDurationFromSettingsOrDefaults("esRefreshWaitTimeout", storeProperties)
	cfg.maxBulkSize, e = getIntFromSettingsOrDefaults("maxBulkSize", storeProperties)
	cfg.maxBulkCount, e = getIntFromSettingsOrDefaults("maxBulkCount", storeProperties)
	// If any error have been encountered, it will be returned
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
