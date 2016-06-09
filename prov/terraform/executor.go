package terraform

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
)

type Executor struct {
}

func (e *Executor) ApplyInfrastructure(id string) error {
	infraPath := path.Join("work", "deployments", fmt.Sprint(id), "infra")
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
func (e *Executor) DestroyInfrastructure(id string) error {
	infraPath := path.Join("work", "deployments", fmt.Sprint(id), "infra")
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
