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

	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/ystia/yorc/config"
	"github.com/ystia/yorc/events"
	"github.com/ystia/yorc/helper/stringutil"
	"github.com/ystia/yorc/prov"
)

type defaultExecutor struct {
	clientset *kubernetes.Clientset
}

func (e *defaultExecutor) ExecAsyncOperation(ctx context.Context, conf config.Configuration, taskID, deploymentID, nodeName string, operation prov.Operation, stepName string) error {
	return errors.New("Asynchronous operation is not yet handled by this executor")
}

func (e *defaultExecutor) ExecOperation(ctx context.Context, conf config.Configuration, taskID, deploymentID, nodeName string, operation prov.Operation) error {
	consulClient, err := conf.GetConsulClient()
	if err != nil {
		return err
	}

	logOptFields, ok := events.FromContext(ctx)
	if !ok {
		return errors.New("Missing contextual log optionnal fields")
	}
	logOptFields[events.NodeID] = nodeName
	logOptFields[events.OperationName] = stringutil.GetLastElement(operation.Name, ".")
	logOptFields[events.InterfaceName] = stringutil.GetAllExceptLastElement(operation.Name, ".")

	ctx = events.NewContext(ctx, logOptFields)

	kv := consulClient.KV()
	exec, err := newExecution(kv, conf, taskID, deploymentID, nodeName, operation)
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
	kubMasterIP := kubConf.GetString("master_url")

	if kubMasterIP == "" {
		return nil, errors.New(`Missing or invalid mandatory parameter master_url in the "kubernetes" infrastructure configuration`)
	}
	conf, err := clientcmd.BuildConfigFromFlags(kubMasterIP, "")
	if err != nil {
		return nil, errors.Wrap(err, "Failed to build kubernetes config")
	}

	conf.TLSClientConfig.Insecure = kubConf.GetBool("insecure")
	conf.TLSClientConfig.CAFile = kubConf.GetString("ca_file")
	conf.TLSClientConfig.CertFile = kubConf.GetString("cert_file")
	conf.TLSClientConfig.KeyFile = kubConf.GetString("key_file")

	clientset, err := kubernetes.NewForConfig(conf)
	return clientset, errors.Wrap(err, "Failed to create kubernetes clientset from config")
}
