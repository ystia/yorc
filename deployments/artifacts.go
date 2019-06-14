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

package deployments

import (
	"path"

	"strings"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"

	"github.com/ystia/yorc/v4/helper/consulutil"
)

// GetArtifactsForType returns a map of artifact name / artifact file for the given type.
//
// The returned artifacts paths are relative to root of the deployment archive.
// It traverse the 'derived_from' relations to support inheritance of artifacts. Parent artifacts are fetched first and may be overridden by child types
func GetArtifactsForType(kv *api.KV, deploymentID, typeName string) (map[string]string, error) {
	parentType, err := GetParentType(kv, deploymentID, typeName)
	if err != nil {
		return nil, err
	}
	var artifacts map[string]string
	if parentType != "" {
		artifacts, err = GetArtifactsForType(kv, deploymentID, parentType)
		if err != nil {
			return nil, err
		}
	} else {
		artifacts = make(map[string]string)
	}
	typePath, err := locateTypePath(kv, deploymentID, typeName)
	if err != nil {
		return nil, err
	}

	artifactsPath := path.Join(typePath, "artifacts")
	importPath, err := GetTypeImportPath(kv, deploymentID, typeName)
	if err != nil {
		return nil, err
	}
	err = updateArtifactsFromPath(kv, artifacts, artifactsPath, importPath)
	return artifacts, errors.Wrapf(err, "Failed to get artifacts for type: %q", typeName)
}

// GetArtifactsForNode returns a map of artifact name / artifact file for the given node.
//
// The returned artifacts paths are relative to root of the deployment archive.
// It will first fetch artifacts from it node type and its parents and fetch artifacts for the node template itself.
// This way artifacts from a parent type may be overridden by child types and artifacts from node type may be overridden by the node template
func GetArtifactsForNode(kv *api.KV, deploymentID, nodeName string) (map[string]string, error) {
	nodeType, err := GetNodeType(kv, deploymentID, nodeName)
	if err != nil {
		return nil, err
	}
	artifacts, err := GetArtifactsForType(kv, deploymentID, nodeType)
	if err != nil {
		return nil, err
	}
	artifactsPath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes", nodeName, "artifacts")
	// No importPath for node templates as they will be CSAR root relative
	err = updateArtifactsFromPath(kv, artifacts, artifactsPath, "")
	return artifacts, errors.Wrapf(err, "Failed to get artifacts for node: %q", nodeName)
}

// updateArtifactsFromPath returns a map of artifact name / artifact file for the given node or type denoted by the given artifactsPath.
func updateArtifactsFromPath(kv *api.KV, artifacts map[string]string, artifactsPath, importPath string) error {
	kvps, _, err := kv.Keys(artifactsPath+"/", "/", nil)
	if err != nil {
		return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}

	for _, artifactPath := range kvps {
		artifactName := path.Base(artifactPath)
		kvp, _, err := kv.Get(path.Join(artifactPath, "file"), nil)
		if err != nil {
			return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		}
		if kvp == nil || len(kvp.Value) == 0 {
			return errors.Errorf("Missing mandatory attribute \"file\" for artifact %q", path.Base(artifactPath))
		}
		// TODO path is relative to the type and may not be the same as a child type
		artifacts[artifactName] = path.Join(importPath, string(kvp.Value))
	}
	return nil
}

// GetArtifactTypeExtensions returns the extensions defined in this artifact type.
// If the artifact doesn't define any extension then a nil slice is returned
func GetArtifactTypeExtensions(kv *api.KV, deploymentID, artifactType string) ([]string, error) {
	typePath, err := locateTypePath(kv, deploymentID, artifactType)
	if err != nil {
		return nil, err
	}
	kvp, _, err := kv.Get(path.Join(typePath, "file_ext"), nil)
	if err != nil {
		return nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if kvp == nil || len(kvp.Value) == 0 {
		return nil, nil
	}
	return strings.Split(string(kvp.Value), ","), nil
}
