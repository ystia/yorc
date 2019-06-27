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

package store

import (
	"context"
	"path"
	"sync"

	"github.com/blang/semver"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"

	"github.com/ystia/yorc/v3/deployments/internal"
	"github.com/ystia/yorc/v3/helper/collections"
	"github.com/ystia/yorc/v3/helper/consulutil"
	"github.com/ystia/yorc/v3/log"
	"github.com/ystia/yorc/v3/tosca"
)

// BuiltinOrigin is the origin for Yorc builtin
const BuiltinOrigin = "builtin"

const yorcOriginConsulKey = "yorc_origin"

var lock sync.Mutex

var builtinTypes = make([]string, 0)

// getLatestCommonsTypesPaths() returns all the path keys corresponding to the last version of a type
// that is stored under the consulutil.CommonsTypesKVPrefix.
// For example, one path key could be _yorc/commons_types/some_type/2.0.0 if the last version
// of the some_type type is 2.0.0
func getLatestCommonsTypesPaths() ([]string, error) {
	kv := consulutil.GetKV()
	// keys, _, err := kv.List("", nil)
	keys, _, err := kv.Keys(consulutil.CommonsTypesKVPrefix+"/", "/", nil)
	if err != nil {
		return nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	paths := make([]string, 0, len(keys))
	for _, builtinTypesPath := range keys {
		versions, _, err := kv.Keys(builtinTypesPath, "/", nil)
		if err != nil {
			return nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		}

		if len(versions) == 0 {
			continue
		}
		var maxVersion semver.Version
		for _, v := range versions {
			version, err := semver.Make(path.Base(v))
			if err == nil && version.GTE(maxVersion) {
				maxVersion = version
			}
		}
		typePath := path.Join(builtinTypesPath, maxVersion.String())

		paths = append(paths, typePath)
	}
	return paths, nil
}

// GetCommonsTypesPaths returns the path of builtin types supported by this instance of Yorc
//
// Returned keys are formatted as <consulutil.CommonsTypesKVPrefix>/<name>/<version>
// If this is used from outside a Yorc instance typically a plugin or another app then the latest
// version of each builtin type stored in Consul is assumed
func GetCommonsTypesPaths() []string {
	lock.Lock()
	defer lock.Unlock()
	if len(builtinTypes) == 0 {
		// Not provided at system startup we are probably in an external application used as a lib
		// So let use latest values of each stored builtin types in Consul
		builtinTypes, _ = getLatestCommonsTypesPaths()
	}
	res := make([]string, len(builtinTypes))
	copy(res, builtinTypes)
	return res
}

// CommonDefinition stores a TOSCA definition to the common place
func CommonDefinition(ctx context.Context, definitionName, origin string, definitionContent []byte) error {
	topology := tosca.Topology{}
	err := yaml.Unmarshal(definitionContent, &topology)
	if err != nil {
		return errors.Wrapf(err, "failed to unmarshal TOSCA definition %q", definitionName)
	}
	ctx, errGroup, consulStore := consulutil.WithContext(ctx)
	kv := consulutil.GetKV()
	name := topology.Metadata["template_name"]
	if name == "" {
		return errors.Errorf("Can't store builtin TOSCA definition %q, template_name is missing", definitionName)
	}
	version := topology.Metadata["template_version"]
	if version == "" {
		return errors.Errorf("Can't store builtin TOSCA definition %q, template_version is missing", definitionName)
	}

	topology.Metadata[yorcOriginConsulKey] = origin

	topologyPrefix := path.Join(consulutil.CommonsTypesKVPrefix, name, version)

	func() {
		lock.Lock()
		defer lock.Unlock()
		if !collections.ContainsString(builtinTypes, topologyPrefix) {
			builtinTypes = append(builtinTypes, topologyPrefix)
		}
	}()

	keys, _, err := kv.Keys(topologyPrefix+"/", "/", nil)
	if err != nil {
		return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if len(keys) > 0 {
		log.Printf("Do not storing existing topology definition: %q version: %q", name, version)
		return nil
	}
	errGroup.Go(func() error {
		internal.StoreTopologyTopLevelKeyNames(ctx, consulStore, topology, topologyPrefix)
		return nil
	})
	errGroup.Go(func() error {
		return internal.StoreRepositories(ctx, consulStore, topology, topologyPrefix)
	})
	errGroup.Go(func() error {
		return internal.StoreAllTypes(ctx, consulStore, topology, topologyPrefix, "")
	})
	return errGroup.Wait()
}

// Deployment stores a whole deployment.
func Deployment(ctx context.Context, topology tosca.Topology, deploymentID, rootDefPath string) error {
	ctx, errGroup, consulStore := consulutil.WithContext(ctx)

	errGroup.Go(func() error {
		return internal.StoreTopology(ctx, consulStore, errGroup, topology, deploymentID, path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology"), "", "", rootDefPath)
	})

	return errGroup.Wait()
}

// Definition is TOSCA Definition registered in the Yorc as builtin could be comming from Yorc itself or a plugin
type Definition struct {
	Name    string `json:"name"`
	Origin  string `json:"origin"`
	Version string `json:"version"`
}

// GetCommonsDefinitionsList returns the list of commons definitions within Yorc
func GetCommonsDefinitionsList() ([]Definition, error) {
	lock.Lock()
	defer lock.Unlock()
	if len(builtinTypes) == 0 {
		// Not provided at system startup we are probably in an external application used as a lib
		// So let use latest values of each stored builtin types in Consul
		builtinTypes, _ = getLatestCommonsTypesPaths()
	}
	kv := consulutil.GetKV()
	res := make([]Definition, len(builtinTypes))
	for _, p := range builtinTypes {
		d := Definition{}
		d.Version = path.Base(p)
		p = path.Dir(p)
		d.Name = path.Base(p)
		res = append(res, d)
		kvp, _, err := kv.Get(path.Join(p, "metadata", yorcOriginConsulKey), nil)
		if err != nil {
			return nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		}
		if kvp != nil {
			d.Version = string(kvp.Value)
		}
	}
	return res, nil
}
