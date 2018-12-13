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

// Variables with an uknown values are initialized in the root Makefile
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
		"infrastructure", "i", "", "Define the type of infrastructure where to deploy Yorc: google, openstack, aws, hostspool")
	viper.BindPFlag("infrastructure", bootstrapCmd.PersistentFlags().Lookup("infrastructure"))
	bootstrapCmd.PersistentFlags().StringVarP(&deploymentType,
		"deployment_type", "d", "single_node", "Define deployment type: single_node or HA")
	viper.BindPFlag("deployment_type", bootstrapCmd.PersistentFlags().Lookup("deployment_type"))
	bootstrapCmd.PersistentFlags().StringVarP(&followType,
		"follow", "f", "steps", "Follow bootstrap deployment steps, logs, or none")
	viper.BindPFlag("follow", bootstrapCmd.PersistentFlags().Lookup("follow"))
	bootstrapCmd.PersistentFlags().StringVarP(&inputsPath,
		"values", "v", "", "Path to file containing input values")
	bootstrapCmd.PersistentFlags().BoolVarP(&reviewInputs, "review", "r", false,
		"Review and update input values before starting the bootstrap")
	viper.BindPFlag("review", bootstrapCmd.PersistentFlags().Lookup("review"))
	bootstrapCmd.PersistentFlags().StringVarP(&resourcesZipFilePath,
		"resources_zip", "z", "", "Path to bootstrap resources zip file")
	viper.BindPFlag("resources_zip", bootstrapCmd.PersistentFlags().Lookup("resources_zip"))
	bootstrapCmd.PersistentFlags().StringVarP(&workingDirectoryPath,
		"working_directory", "w", "work", "Working directory where to place deployment files")
	viper.BindPFlag("working_directory", bootstrapCmd.PersistentFlags().Lookup("working_directory"))

	viper.SetEnvPrefix(commands.EnvironmentVariablePrefix)
	viper.AutomaticEnv() // read in environment variables that match
	viper.BindEnv("infrastructure")
	viper.BindEnv("deployment_type")
	viper.BindEnv("follow")
	viper.BindEnv("review")
	viper.BindEnv("resources_zip")
	viper.BindEnv("working_directory")

	// Adding extra parameters for bootstrap configuration values
	args := os.Args
	setBootstrapExtraParams(args, bootstrapCmd)

	cleanCmd := &cobra.Command{
		Use:   "cleanup",
		Short: "Cleans local setup",
		Long:  `Cleans local setup, without undeploying the bootstrapped Yorc.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cleanBootstrapSetup(workingDirectoryPath)
		},
	}
	bootstrapCmd.AddCommand(cleanCmd)

	// Adding flags related to the infrastructure
	commands.InitExtraFlags(args, bootstrapCmd)

	// When review is set through an environment variable
	// it has to be converted as a boolean or it is not taken into account
	// correctly
	reviewInputs = viper.GetBool("review")

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

	// Read config if any
	viper.SetConfigName("config.yorc") // name of config file (without extension)
	viper.AddConfigPath(workingDirectoryPath)
	viper.ReadInConfig()

	configuration := commands.GetConfig()

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
