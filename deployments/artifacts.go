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
)

func updateArtifactsForType(ctx context.Context, deploymentID, typeName, importPath string, artifacts map[string]string) error {
	artifactsMap, err := getTypeArtifacts(deploymentID, typeName)
	if err != nil {
		return nil
	}

	for k, v := range artifactsMap {
		if v.File != "" {
			// TODO path is relative to the type and may not be the same as a child type
			artifacts[k] = path.Join(importPath, v.File)
		}
	}

	return nil
}

// GetFileArtifactsForType returns a map of artifact name / artifact file for the given type of type tType
//
// The returned artifacts paths are relative to root of the deployment archive.
// It traverse the 'derived_from' relations to support inheritance of artifacts. Parent artifacts are fetched first and may be overridden by child types
func GetFileArtifactsForType(ctx context.Context, deploymentID, typeName  string) (map[string]string, error) {
	parentType, err := GetParentType(ctx, deploymentID, typeName)
	if err != nil {
		return nil, err
	}
	var artifacts map[string]string
	if parentType != "" {
		artifacts, err = GetFileArtifactsForType(ctx, deploymentID, parentType)
		if err != nil {
			return nil, err
		}
	} else {
		artifacts = make(map[string]string)
	}

	importPath, err := GetTypeImportPath(ctx, deploymentID, typeName)
	if err != nil {
		return nil, err
	}
	err = updateArtifactsForType(ctx, deploymentID, typeName, importPath, artifacts)
	return artifacts, errors.Wrapf(err, "Failed to get artifacts for type: %q", typeName)
}

// GetFileArtifactsForNode returns a map of artifact name / artifact file for the given node.
//
// The returned artifacts paths are relative to root of the deployment archive.
// It will first fetch artifacts from it node type and its parents and fetch artifacts for the node template itself.
// This way artifacts from a parent type may be overridden by child types and artifacts from node type may be overridden by the node template
func GetFileArtifactsForNode(ctx context.Context, deploymentID, nodeName string) (map[string]string, error) {
	node, err := getNodeTemplateStruct(ctx, deploymentID, nodeName)
	if err != nil {
		return nil, err
	}
	artifacts, err := GetFileArtifactsForType(ctx, deploymentID, node.Type)
	if err != nil {
		return nil, err
	}

	// No importPath for node templates as they will be CSAR root relative
	for k, v := range node.Artifacts {
		if v.File != "" {
			artifacts[k] = v.File
		}
	}
	return artifacts, errors.Wrapf(err, "Failed to get artifacts for node: %q", nodeName)
}

// GetArtifactTypeExtensions returns the extensions defined in this artifact type.
// If the artifact doesn't define any extension then a nil slice is returned
func GetArtifactTypeExtensions(ctx context.Context, deploymentID, artifactTypeName string) ([]string, error) {
	artifactTyp := new(tosca.ArtifactType)
	err := getTypeStruct(deploymentID, artifactTypeName, artifactTyp)
	if err != nil {
		return nil, err
	}
	return artifactTyp.FileExt, nil
}
