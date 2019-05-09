// Copyright 2018 Bull S.A.S. Atos Technologies - Bull, Rue Jean Jaures, B.P.68, 78340, Les Clayes-sous-Bois, France.
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

package resources

import (
	"context"
	"io/ioutil"
	"path"

	"github.com/pkg/errors"
	res "gopkg.in/cookieo9/resources-go.v2"

	"github.com/ystia/yorc/v3/deployments/store"
	"github.com/ystia/yorc/v3/log"
)

// StoreBuiltinTOSCAResources retrieves TOSCA definition within the Yorc binary and register them into Consul
//
// Note that the consulpublisher should be initialized with a KV before running this function
func StoreBuiltinTOSCAResources() error {
	resources, err := getToscaResources()
	if err != nil {
		log.Panicf("Failed to load builtin Tosca definition. %v", err)
	}
	ctx := context.Background()
	for defName, defContent := range resources {
		err = store.CommonDefinition(ctx, defName, store.BuiltinOrigin, defContent)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetTOSCADefinition allows to retrieve a TOSCA definition stored into Yorc executable
func GetTOSCADefinition(name string) ([]byte, error) {
	// try to get resources from Yorc executable
	// Use the embedded resources
	exePath, err := res.ExecutablePath()
	if err != nil {
		return nil, err
	}
	bundle, err := res.OpenZip(exePath)
	if err != nil {
		return nil, errors.Wrap(err, "can't find TOSCA resources in yorc executable, not a valid zip file")
	}
	defer bundle.Close()
	searcher := bundle.(res.Searcher)
	toscaRes, err := searcher.Find("tosca/" + name)

	return ReadResource(toscaRes)
}

// ReadResource is an helper function that reads bytes from a Resource
func ReadResource(r res.Resource) ([]byte, error) {
	// Use path instead of filepath as it is platform independent
	rName := path.Base(r.Path())
	reader, err := r.Open()
	if err != nil {
		return nil, errors.Wrapf(err, "can't open local TOSCA resource %s", rName)
	}
	defer reader.Close()
	rContent, err := ioutil.ReadAll(reader)
	return rContent, errors.Wrapf(err, "can't read local TOSCA resource %s", rName)
}

// ReadResources is an helper function that reads bytes from a slice of Resource
//
// Resources are indexed based on there base path name
func ReadResources(rs []res.Resource) (map[string][]byte, error) {
	res := make(map[string][]byte, len(rs))
	for _, r := range rs {
		rContent, err := ReadResource(r)
		if err != nil {
			return nil, err
		}
		res[path.Base(r.Path())] = rContent
	}
	return res, nil
}

func getToscaResources() (map[string][]byte, error) {
	// try to get resources from Yorc executable
	// Use the embedded resources
	exePath, err := res.ExecutablePath()
	if err != nil {
		return nil, err
	}
	bundle, err := res.OpenZip(exePath)
	if err != nil {
		return nil, errors.Wrap(err, "can't find TOSCA resources in yorc executable, not a valid zip file")
	}
	defer bundle.Close()
	searcher := bundle.(res.Searcher)
	toscaResources, err := searcher.Glob("tosca/*.yml")
	if err != nil {
		return nil, errors.Wrap(err, "can't find TOSCA resources in yorc executable")
	}

	return ReadResources(toscaResources)
}
