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

package kubernetes

import (
	"context"
	"os"
	"time"

	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	// Loading the gcp plugin to authenticate against GKE clusters
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	"github.com/ystia/yorc/config"
	"github.com/ystia/yorc/helper/stringutil"
	"github.com/ystia/yorc/log"
	"github.com/ystia/yorc/prov"
)

type defaultExecutor struct {
	clientset kubernetes.Interface
}

func getExecution(conf config.Configuration, taskID, deploymentID, nodeName string, operation prov.Operation) (*execution, error) {
	consulClient, err := conf.GetConsulClient()
	if err != nil {
		return nil, err
	}

	kv := consulClient.KV()
	return newExecution(kv, conf, taskID, deploymentID, nodeName, operation)
}

func (e *defaultExecutor) ExecAsyncOperation(ctx context.Context, conf config.Configuration, taskID, deploymentID, nodeName string, operation prov.Operation, stepName string) (*prov.Action, time.Duration, error) {
	exec, err := getExecution(conf, taskID, deploymentID, nodeName, operation)
	if err != nil {
		return nil, 0, err
	}

	if e.clientset == nil {
		e.clientset, err = initClientSet(conf)
		if err != nil {
			return nil, 0, err
		}
	}

	return exec.executeAsync(ctx, conf, stepName, e.clientset)
}

func (e *defaultExecutor) ExecOperation(ctx context.Context, conf config.Configuration, taskID, deploymentID, nodeName string, operation prov.Operation) error {
	exec, err := getExecution(conf, taskID, deploymentID, nodeName, operation)
	if err != nil {
		return err
	}

	if e.clientset == nil {
		e.clientset, err = initClientSet(conf)
		if err != nil {
			return err
		}
	}

	return exec.execute(ctx, e.clientset)
}

func initClientSet(cfg config.Configuration) (*kubernetes.Clientset, error) {
	kubConf := cfg.Infrastructures["kubernetes"]

	var conf *rest.Config
	var err error
	var kubeMasterIP string
	var kubeConfigPathOrContent string

	if kubConf != nil {
		kubeMasterIP = kubConf.GetString("master_url")
		kubeConfigPathOrContent = kubConf.GetString("kubeconfig")
	}

	if kubeConfigPathOrContent == "" && kubeMasterIP == "" {
		// No details on the kubernetes cluster in orchestrator configuration.
		// Considering this is a configuration where the orchestrator runs within
		// the Kubernetes Cluster
		log.Debugf("No Kubernetes cluster specified in configuration, attempting to authenticate inside the cluster")
		conf, err = rest.InClusterConfig()
		if err != nil {
			return nil, errors.Wrap(err, "Failed to build kubernetes InClusterConfig")
		}
	} else {

		// kubeconfig yorc infrastructure config parameter is either a path to a
		// file or a string providing the Kubernetes configuration details.
		// The kubernetes go API expects to get a file, creating it if necessary
		var kubeConfigPath string
		var wasPath bool
		if kubeConfigPathOrContent != "" {
			if kubeConfigPath, wasPath, err = stringutil.GetFilePath(kubeConfigPathOrContent); err != nil {
				return nil, errors.Wrap(err, "Failed to get Kubernetes config file")
			}
			if !wasPath {
				defer os.Remove(kubeConfigPath)
			}
		}

		// Google Application credentials are needed when attempting to access
		// Kubernetes Engine outside the Kubernetes cluster from a host where gcloud
		// SDK is not installed.
		// The application_credentials yorc infrastructure config parameter is
		// either a path to a file or a string providing the credentials.
		// Google API expects a path to a file to be provided in an environment
		// variable GOOGLE_APPLICATION_CREDENTIALS.
		// So creating a file if necessary

		applicationCredsPathOrContent := kubConf.GetString("application_credentials")
		if applicationCredsPathOrContent != "" {

			applicationCredsPath, wasPath, err := stringutil.GetFilePath(applicationCredsPathOrContent)
			if err != nil {
				return nil, errors.Wrap(err, "Failed to get application credentials file")
			}
			if !wasPath {
				defer os.Remove(applicationCredsPath)
			}

			os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", applicationCredsPath)
			defer os.Unsetenv("GOOGLE_APPLICATION_CREDENTIALS")
		}

		conf, err = clientcmd.BuildConfigFromFlags(kubeMasterIP, kubeConfigPath)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to build kubernetes config")
		}

		if kubeConfigPath == "" {
			conf.TLSClientConfig.Insecure = kubConf.GetBool("insecure")
			conf.TLSClientConfig.CAFile = kubConf.GetString("ca_file")
			conf.TLSClientConfig.CertFile = kubConf.GetString("cert_file")
			conf.TLSClientConfig.KeyFile = kubConf.GetString("key_file")
		} else if conf.AuthProvider != nil && conf.AuthProvider.Name == "gcp" {
			// When application credentials are set, using these creds
			// and not attempting to rely on the local host gcloud command to
			// access tokens, as gcloud may not be installed on yorc host
			delete(conf.AuthProvider.Config, "cmd-path")
		}
	}

	clientset, err := kubernetes.NewForConfig(conf)
	return clientset, errors.Wrap(err, "Failed to create kubernetes clientset from config")
}
