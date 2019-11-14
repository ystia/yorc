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
	"context"
	"github.com/ystia/yorc/v4/tosca"
	"path"

	"github.com/pkg/errors"

	"github.com/ystia/yorc/v4/helper/consulutil"
)

// GetArtifactsForType returns a map of artifact name / artifact file for the given type.
//
// The returned artifacts paths are relative to root of the deployment archive.
// It traverse the 'derived_from' relations to support inheritance of artifacts. Parent artifacts are fetched first and may be overridden by child types
func GetArtifactsForType(ctx context.Context, deploymentID, typeName string) (map[string]string, error) {
	parentType, err := GetParentType(ctx, deploymentID, typeName)
	if err != nil {
		return nil, err
	}
	var artifacts map[string]string
	if parentType != "" {
		artifacts, err = GetArtifactsForType(ctx, deploymentID, parentType)
		if err != nil {
			return nil, err
		}
	} else {
		artifacts = make(map[string]string)
	}
	typePath, err := locateTypePath(deploymentID, typeName)
	if err != nil {
		return nil, err
	}

	artifactsPath := path.Join(typePath, "artifacts")
	importPath, err := GetTypeImportPath(ctx, deploymentID, typeName)
	if err != nil {
		return nil, err
	}
	err = updateArtifactsFromPath(artifacts, artifactsPath, importPath)
	return artifacts, errors.Wrapf(err, "Failed to get artifacts for type: %q", typeName)
}

// GetArtifactsForNode returns a map of artifact name / artifact file for the given node.
//
// The returned artifacts paths are relative to root of the deployment archive.
// It will first fetch artifacts from it node type and its parents and fetch artifacts for the node template itself.
// This way artifacts from a parent type may be overridden by child types and artifacts from node type may be overridden by the node template
func GetArtifactsForNode(ctx context.Context, deploymentID, nodeName string) (map[string]string, error) {
	nodeType, err := GetNodeType(ctx, deploymentID, nodeName)
	if err != nil {
		return nil, err
	}
	artifacts, err := GetArtifactsForType(ctx, deploymentID, nodeType)
	if err != nil {
		return nil, err
	}
	artifactsPath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes", nodeName, "artifacts")
	// No importPath for node templates as they will be CSAR root relative
	err = updateArtifactsFromPath(artifacts, artifactsPath, "")
	return artifacts, errors.Wrapf(err, "Failed to get artifacts for node: %q", nodeName)
}

// updateArtifactsFromPath returns a map of artifact name / artifact file for the given node or type denoted by the given artifactsPath.
func updateArtifactsFromPath(artifacts map[string]string, artifactsPath, importPath string) error {
	keys, err := consulutil.GetKeys(artifactsPath)
	if err != nil {
		return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}

	for _, artifactPath := range keys {
		artifactName := path.Base(artifactPath)
		exist, value, err := consulutil.GetStringValue(path.Join(artifactPath, "file"))
		if err != nil {
			return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		}
		if !exist || value == "" {
			return errors.Errorf("Missing mandatory attribute \"file\" for artifact %q", path.Base(artifactPath))
		}
		// TODO path is relative to the type and may not be the same as a child type
		artifacts[artifactName] = path.Join(importPath, value)
	}
	return nil
}

// GetArtifactTypeExtensions returns the extensions defined in this artifact type.
// If the artifact doesn't define any extension then a nil slice is returned
func GetArtifactTypeExtensions(ctx context.Context, deploymentID, artifactType string) ([]string, error) {
	artifactTyp := new(tosca.ArtifactType)
	exist, err := getType(deploymentID, artifactType, artifactTyp)
	if err != nil {
		return nil, err
	}
	if !exist {
		return nil, nil
	}

	return artifactTyp.FileExt, nil
}
