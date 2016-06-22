package terraform

import (
	"log"
	"os"
	"os/exec"
	"path"
)

type Executor struct {
}

func (e *Executor) ApplyInfrastructure(depId, nodeName string) error {
	infraPath := path.Join("work", "deployments", depId, "infra", nodeName)
	cmd := exec.Command("terraform", "apply")
	cmd.Dir = infraPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		log.Print(err)
		return err
	}

	return cmd.Wait()

}
func (e *Executor) DestroyInfrastructure(depId, nodeName string) error {
	infraPath := path.Join("work", "deployments", depId, "infra", nodeName)
	cmd := exec.Command("terraform", "destroy", "-force")
	cmd.Dir = infraPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		log.Print(err)
		return err
	}

	return cmd.Wait()

}
