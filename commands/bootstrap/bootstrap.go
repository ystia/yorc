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

// Variables with an uknown values are initialized in the root Makefile
var (
	alien4cloudVersion = getAlien4CloudVersionFromTOSCATypes()
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
		"location", "l", "", "Define the type of location where to deploy Yorc")
	bootstrapCmd.PersistentFlags().StringVarP(&deploymentType,
		"deployment_type", "d", "single_node", "Define deployment type: single_node or HA")
	bootstrapCmd.PersistentFlags().StringVarP(&inputsPath,
		"inputs", "i", "", "Path to inputs file")
	bootstrapCmd.PersistentFlags().StringVarP(&followType,
		"follow", "f", "steps", "Follow bootstrap deployment steps, logs, or none")
	bootstrapCmd.PersistentFlags().StringVarP(&resourcesZipFilePath,
		"resources", "r", "", "Path to bootstrap resources zip file")
	bootstrapCmd.PersistentFlags().StringVarP(&workingDirectoryPath,
		"working_directory", "w", "work", "Working directory where to place deployment files")
	bootstrapCmd.PersistentFlags().BoolVarP(&reviewInputs, "review_inputs", "u", false,
		"Review and update inputs before starting the bootstrap")

	// Adding flags and environment variables for inputs having default values
	setDefaultInputValues()

	cleanCmd := &cobra.Command{
		Use:   "cleanup",
		Short: "Cleans local setup",
		Long:  `Cleans local setup, without undeploying the bootstrapped Yorc.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cleanBootstrapSetup(workingDirectoryPath)
		},
	}
	bootstrapCmd.AddCommand(cleanCmd)

	args := os.Args

	commands.ServerInitExtraFlags(args)

	viper.SetEnvPrefix(environmentVariablePrefix)
	viper.AutomaticEnv()               // read in environment variables that match
	viper.SetConfigName("config.yorc") // name of config file (without extension)
	viper.AddConfigPath("work")        // adding default work directory as first search path
	viper.AddConfigPath(".")
}

var bootstrapCmd *cobra.Command
var bootstrapViper *viper.Viper
var infrastructureType string
var deploymentType string
var followType string
var reviewInputs bool
var resourcesZipFilePath string
var workingDirectoryPath string
var inputsPath string

func bootstrap() error {

	infrastructureType = strings.ToLower(infrastructureType)
	deploymentType = strings.ToLower(deploymentType)

	// Resources, like the topology zip, will be created in a directory under
	//the working directory
	resourcesDir := filepath.Join(workingDirectoryPath, "bootstrapResources")
	if err := os.RemoveAll(resourcesDir); err != nil {
		return err
	}

	// First extract resources files
	if err := extractResources(resourcesZipFilePath, resourcesDir); err != nil {
		return err
	}

	configuration := commands.GetConfig()
	fmt.Printf("LOLO config : %+v\n", configuration)
	// Initializing parameters from environment variables, CLI options
	// input file, and asking for user input if needed
	if err := initializeInputs(inputsPath, resourcesDir, configuration); err != nil {
		return err
	}

	// Create a topology from resources provided in the resources directory
	topologyDir := filepath.Join(resourcesDir, "topology")
	if err := createTopology(topologyDir); err != nil {
		return err
	}

	// Now that the topology is ready, it needs a local Yorc server able to
	// deploy it
	if err := setupYorcServer(workingDirectoryPath); err != nil {
		return err
	}

	// A local Yorc server is running, using it to deploy the topology
	deploymentID, errDeploy := deployTopology(workingDirectoryPath, topologyDir)

	if errDeploy != nil {
		return errDeploy
	}

	if err := followDeployment(deploymentID, followType); err != nil {
		return err
	}

	//	err = tearDownYorcServer(workingDirectoryPath)
	return nil
}
