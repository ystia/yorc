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
	"testing"

	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/require"
	"github.com/ystia/yorc/v3/testutil"
)

func testRepositories(t *testing.T, kv *api.KV) {

	deploymentID := testutil.BuildDeploymentID(t)
	err := StoreDeploymentDefinition(context.Background(), kv, deploymentID, "testdata/test_repositories.yaml")
	require.Nil(t, err, "Failed to parse testdata/test_repositories.yaml definition")

	t.Run("GetRepositoryURLFromName", func(t *testing.T) {
		testGetRepositoryURLFromName(t, kv, deploymentID)
	})
	t.Run("GetRepositoryTokenTypeFromName", func(t *testing.T) {
		testGetRepositoryTokenTypeFromName(t, kv, deploymentID)
	})
	t.Run("GetRepositoryTokenUserFromName", func(t *testing.T) {
		testGetRepositoryTokenUserFromName(t, kv, deploymentID)
	})
}

func testGetRepositoryURLFromName(t *testing.T, kv *api.KV, deploymentID string) {
	tests := []struct {
		name     string
		repoName string
		wantURL  string
		wantErr  bool
	}{
		{"DockerRepo", "ystia-artifactory-docker", "ystia-docker.jfrog.io", false},
		{"HTTPRepo", "ystia-artifactory-http", "https://ystia.jfrog.io/ystia/binaries/", false},
		{"MissingRepo", "ystia-artifactory-other", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotURL, err := GetRepositoryURLFromName(kv, deploymentID, tt.repoName)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetRepositoryURLFromName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotURL != tt.wantURL {
				t.Errorf("GetRepositoryURLFromName() = %v, want %v", gotURL, tt.wantURL)
			}
		})
	}
}

func testGetRepositoryTokenTypeFromName(t *testing.T, kv *api.KV, deploymentID string) {
	tests := []struct {
		name     string
		repoName string
		want     string
		wantErr  bool
	}{
		{"DockerRepo", "ystia-artifactory-docker", "password_token", false},
		{"HTTPRepo", "ystia-artifactory-http", "password", false},
		{"MissingRepo", "ystia-artifactory-other", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetRepositoryTokenTypeFromName(kv, deploymentID, tt.repoName)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetRepositoryTokenTypeFromName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetRepositoryTokenTypeFromName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func testGetRepositoryTokenUserFromName(t *testing.T, kv *api.KV, deploymentID string) {
	tests := []struct {
		name      string
		repoName  string
		wantToken string
		wantUser  string
		wantErr   bool
	}{
		{"DockerRepo", "ystia-artifactory-docker", "my_super_secret_passwd", "myuser", false},
		{"HTTPRepo", "ystia-artifactory-http", "my_super_secret_passwd", "myuser", false},
		{"MissingRepo", "ystia-artifactory-other", "", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotToken, gotUser, err := GetRepositoryTokenUserFromName(kv, deploymentID, tt.repoName)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetRepositoryTokenUserFromName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotToken != tt.wantToken {
				t.Errorf("GetRepositoryTokenUserFromName() gotToken = %v, want %v", gotToken, tt.wantToken)
			}
			if gotUser != tt.wantUser {
				t.Errorf("GetRepositoryTokenUserFromName() gotUser = %v, want %v", gotUser, tt.wantUser)
			}
		})
	}
}
