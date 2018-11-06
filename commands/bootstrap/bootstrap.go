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

package bootstrap

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ystia/yorc/commands"
)

const (
	environmentVariablePrefix = "YORC_BOOTSTRAP"
)

// Variables initialized in the root Makefile
var (
	alien4cloudVersion = "unknown"
	ansibleVersion     = "unknown"
	consulVersion      = "unknown"
	terraformVersion   = "unknown"
	yorcVersion        = "unknown"
)

func init() {

	// bootstrapViper is the viper configuration for bootstrap command and its children
	bootstrapViper = viper.New()

	// bootstrapCmd is the bootstrap base command
	bootstrapCmd = &cobra.Command{
		Use:           "bootstrap",
		Aliases:       []string{"boot", "b"},
		Short:         "Installs Yorc and its dependencies",
		Long:          `Installs Yorc and its dependencies on a given infrastructure`,
		SilenceErrors: true,
		Run: func(cmd *cobra.Command, args []string) {
			err := bootstrap()
			if err != nil {
				fmt.Println("Bootstrap error", err)
			}
		},
	}

	commands.RootCmd.AddCommand(bootstrapCmd)
	bootstrapCmd.PersistentFlags().StringVarP(&infrastructureType,
		"location", "l", "openstack", "Define the type of location where to deploy Yorc")
	bootstrapCmd.PersistentFlags().StringVarP(&deploymentType,
		"deployment-type", "d", "single-node", "Define deployment type: single-node or HA")
	bootstrapCmd.PersistentFlags().StringVarP(&inputsPath,
		"inputs", "i", "", "Path to inputs file")
	bootstrapCmd.PersistentFlags().StringVarP(&topologyZipPath,
		"topology", "t", "", "Path to topology zip file")
	bootstrapCmd.PersistentFlags().StringVarP(&workingDirectoryPath,
		"working-directory", "w", "work", "Working directory where to place deployment files")

	// Adding flags and environment variables for inputs having default values
	setDefaultInputValues()
}

var bootstrapCmd *cobra.Command
var bootstrapViper *viper.Viper
var infrastructureType string
var deploymentType string
var topologyZipPath string
var workingDirectoryPath string
var inputsPath string

func bootstrap() error {

	infrastructureType = strings.ToLower(infrastructureType)
	deploymentType = strings.ToLower(deploymentType)

	// The topology will be created in a directory under the working
	// directory
	topologyPath := filepath.Join(workingDirectoryPath, "bootstrapYorcTopoloy")
	if err := os.RemoveAll(topologyPath); err != nil {
		return err
	}

	if err := createTopology(topologyZipPath, topologyPath, inputsPath); err != nil {
		return err
	}

	// Now that the topology is ready, it needs a local Yorc server able to
	// deploy it
	if err := setupYorcServer(workingDirectoryPath); err != nil {
		return err
	}

	// A local Yorc server is running, using it to deploythe topology
	deploymentID, errDeploy := deployTopology(topologyPath)

	if errDeploy != nil {
		return errDeploy
	}

	if err := followDeployment(deploymentID); err != nil {
		return err
	}

	//	err = tearDownYorcServer(workingDirectoryPath)
	return nil
}
