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

package plugin

import (
	stdlog "log"
	"os"

	hclog "github.com/hashicorp/go-hclog"
	"github.com/ystia/yorc/v3/log"
)

// Creating a hclog logger so that Hashicorp go plugin logs can be correctly
// parsed by Yorc Server parent process, and filtered appropriately,
// according to the parent process log level.
var hclogger = hclog.New(&hclog.LoggerOptions{
	// Plugin logs have to go to stderr in order to be shown in the parent process
	Output: os.Stderr,
	Level:  hclog.Debug,
	// Parent process expect child log in JSON format to be able to parse them
	JSONFormat: true,
})

// Writer parsing log messages to infer the log level from message prefix
// [DEBUG], [INFO], [WARN], [ERROR]
var stdWriter = hclogger.StandardWriter(&hclog.StandardLoggerOptions{InferLevels: true})

// initPluginStdLog configures standard logs to use hclog in the plugin
func initPluginStdLog() {
	stdlog.SetOutput(stdWriter)
	stdlog.SetPrefix("")
	stdlog.SetFlags(0)
}

// initPluginYorcLog configures Yorc log to use hclog in the plugin
func initPluginYorcLog() {
	log.SetOutput(stdWriter)
	log.SetPrefix("")
	log.SetFlags(0)
}
