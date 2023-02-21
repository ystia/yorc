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

package aws

import (
	"context"

	"github.com/pkg/errors"
	"github.com/ystia/yorc/v4/deployments"
)

func verifyThatNodeIsTypeOf(ctx context.Context, nodeParams nodeParams, nodeType string) error {
	res, err := deployments.GetNodeType(ctx, nodeParams.deploymentID, nodeParams.nodeName)
	if err != nil {
		return err
	}
	if res != nodeType {
		return errors.Errorf("Unsupported node type for %q: %s", nodeParams.nodeName, nodeType)
	}
	return nil
}

func getTagsMap(ctx context.Context, nodeParams nodeParams, nestedKeys ...string) (map[string]string, error) {
	tagsVal, err := deployments.GetNodePropertyValue(ctx, nodeParams.deploymentID, nodeParams.nodeName, "tags")
	if err != nil {
		return nil, err
	}
	var tags map[string]string
	if tagsVal != nil && tagsVal.RawString() != "" {
		d, ok := tagsVal.Value.(map[string]interface{})
		if !ok {
			return nil, errors.New("failed to retrieve tags map from Tosca Value: not expected type")
		}

		tags = make(map[string]string, len(d))
		for k, v := range d {
			v, ok := v.(string)
			if !ok {
				return nil, errors.Errorf("failed to retrieve string value from tags map from Tosca Value:%q not expected type", v)
			}
			tags[k] = v
		}
	}

	return tags, nil
}
