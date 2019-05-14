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

package plugin

import (
	"os"
	"os/exec"

	hclog "github.com/hashicorp/go-hclog"
	gplugin "github.com/hashicorp/go-plugin"
	"github.com/ystia/yorc/v3/log"
)

// NewClient returns a properly configured plugin client for a given plugin path
func NewClient(pluginPath string) *gplugin.Client {

	var logLevel hclog.Level
	if log.IsDebug() {
		logLevel = hclog.Debug
	} else {
		logLevel = hclog.Info
	}

	// Create an hclog.Logger to be able to infer plugin logs level
	logger := hclog.New(&hclog.LoggerOptions{
		Output: os.Stdout,
		Level:  logLevel,
		// Using the same time format as standard logs, instead of hclog default
		// format which is a version of RFC3339
		TimeFormat: "2006/01/02 15:04:05",
	})

	// Setup RPC communication
	SetupPluginCommunication()

	return gplugin.NewClient(&gplugin.ClientConfig{
		HandshakeConfig: HandshakeConfig,
		Plugins:         getPlugins(nil),
		Cmd:             exec.Command(pluginPath),
		Logger:          logger,
	})
}
