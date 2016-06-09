package tasks

import (
	"github.com/hashicorp/consul/api"
	"log"
	"novaforge.bull.com/starlings-janus/janus/prov/terraform"
	"novaforge.bull.com/starlings-janus/janus/prov/terraform/openstack"
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
					osGenerator := openstack.NewGenerator(w.consulClient)
					if err := osGenerator.GenerateTerraformInfra(task.TargetId); err != nil {
						task.WithStatus("failed")
						log.Printf("%v. Aborting", err)
						continue

					}
					executor := &terraform.Executor{}
					if err := executor.ApplyInfrastructure(task.TargetId); err != nil {
						task.WithStatus("failed")
						log.Printf("%v. Aborting", err)
						continue
					}
				case UNDEPLOY:
					// TODO as it may be done on another instance we should not expect that infrastructure description is stored locally
					executor := &terraform.Executor{}
					if err := executor.DestroyInfrastructure(task.TargetId); err != nil {
						task.WithStatus("failed")
						log.Printf("%v. Aborting", err)
						continue
					}
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
