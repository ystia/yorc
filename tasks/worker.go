package tasks

import (
	"fmt"
	"github.com/hashicorp/consul/api"
	"log"
	"novaforge.bull.com/starlings-janus/janus/deployments"
	"novaforge.bull.com/starlings-janus/janus/prov/terraform"
	"novaforge.bull.com/starlings-janus/janus/prov/terraform/openstack"
	"strings"
)

type Worker struct {
	workerPool   chan chan *Task
	TaskChannel  chan *Task
	shutdownCh   chan struct{}
	consulClient *api.Client
}

func NewWorker(workerPool chan chan *Task, shutdownCh chan struct{}, consulClient *api.Client) Worker {
	return Worker{
		workerPool:  workerPool,
		TaskChannel: make(chan *Task),
		shutdownCh:  shutdownCh, consulClient: consulClient}
}

func (w Worker) setDeploymentStatus(deploymentId string, status deployments.DeploymentStatus) {
	p := &api.KVPair{Key: strings.Join([]string{deployments.DeploymentKVPrefix, deploymentId, "status"}, "/"), Value: []byte(fmt.Sprint(status))}
	kv := w.consulClient.KV()
	kv.Put(p, nil)
}

// Start method starts the run loop for the worker, listening for a quit channel in
// case we need to stop it
func (w Worker) Start() {
	go func() {
		for {
			// register the current worker into the worker queue.
			w.workerPool <- w.TaskChannel

			select {
			case task := <-w.TaskChannel:
				// we have received a work request.
				log.Printf("Worker got task with id %s", task.Id)
				switch task.TaskType {

				case DEPLOY:
					w.setDeploymentStatus(task.TargetId, deployments.DEPLOYMENT_IN_PROGRESS)
					osGenerator := openstack.NewGenerator(w.consulClient)
					if err := osGenerator.GenerateTerraformInfra(task.TargetId); err != nil {
						task.WithStatus("failed")
						log.Printf("%v. Aborting", err)
						w.setDeploymentStatus(task.TargetId, deployments.DEPLOYMENT_FAILED)
						continue

					}
					executor := &terraform.Executor{}
					if err := executor.ApplyInfrastructure(task.TargetId); err != nil {
						task.WithStatus("failed")
						log.Printf("%v. Aborting", err)
						w.setDeploymentStatus(task.TargetId, deployments.DEPLOYMENT_FAILED)
						continue
					}
					w.setDeploymentStatus(task.TargetId, deployments.DEPLOYED)
				case UNDEPLOY:
					// TODO as it may be done on another instance we should not expect that infrastructure description is stored locally
					w.setDeploymentStatus(task.TargetId, deployments.UNDEPLOYMENT_IN_PROGRESS)
					executor := &terraform.Executor{}
					if err := executor.DestroyInfrastructure(task.TargetId); err != nil {
						task.WithStatus("failed")
						log.Printf("%v. Aborting", err)
						w.setDeploymentStatus(task.TargetId, deployments.UNDEPLOYMENT_FAILED)
						continue
					}
					w.setDeploymentStatus(task.TargetId, deployments.UNDEPLOYED)
				}
				task.WithStatus("done")
				task.TaskLock.Unlock()
				task.TaskLock.Destroy()

			case <-w.shutdownCh:
				// we have received a signal to stop
				log.Printf("Worker received shutdown signal. Exiting...")
				return
			}
		}
	}()
}
