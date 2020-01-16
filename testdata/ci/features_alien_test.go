package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/alien4cloud/alien4cloud-go-client/v2/alien4cloud"
)

func (c *suiteContext) iHaveUploadedTheArtifactNamedToAlien(csar string) error {
	fp := filepath.Join("components", "*"+csar+"*.zip")
	matches, err := filepath.Glob(fp)
	if err != nil {
		return fmt.Errorf("failed to find CSARs matching %q: %w", fp, err)
	}
	if len(matches) == 0 {
		return fmt.Errorf("no CSAR matching %q", fp)
	}
	for _, match := range matches {
		f, err := os.Open(match)
		if err != nil {
			return fmt.Errorf("failed to open CSAR %q: %w", match, err)
		}
		_, err = c.a4cClient.CatalogService().UploadCSAR(c.ctx, f, "")
		if err != nil {
			var pErr alien4cloud.ParsingErr
			if !errors.As(err, &pErr) || pErr.HasCriticalErrors() {
				return fmt.Errorf("failed to upload CSAR %q: %w", match, err)
			}
		}
	}
	return nil
}

func (c *suiteContext) iHaveCreatedAnApplicationNamedBasedOnTemplateNamed(appName, templateName string) error {
	var err error
	c.applicationID, err = c.a4cClient.ApplicationService().CreateAppli(c.ctx, appName, templateName)
	if err != nil {
		return fmt.Errorf("failed to create application %q base on template %q: %w", appName, templateName, err)
	}

	c.environmentID, err = c.a4cClient.ApplicationService().GetEnvironmentIDbyName(c.ctx, c.applicationID, alien4cloud.DefaultEnvironmentName)
	if err != nil {
		return fmt.Errorf("failed to get environment id for application %q: %w", appName, err)
	}
	return nil
}

func (c *suiteContext) iDeployTheApplicationNamed(appName string) error {
	if c.applicationID == "" || c.environmentID == "" {
		return fmt.Errorf("Missing mandatory context parameter applicationID: %q or environmentID: %q", c.applicationID, c.environmentID)
	}
	// use a specific timeout for deploy
	ctx, cancel := context.WithTimeout(c.ctx, 60*time.Minute)
	defer cancel()
	err := c.a4cClient.DeploymentService().DeployApplication(ctx, c.applicationID, c.environmentID, "")
	if err != nil {
		return fmt.Errorf("failed to deploy application %q: %w", appName, err)
	}
	state, err := c.a4cClient.DeploymentService().WaitUntilStateIs(ctx, c.applicationID, c.environmentID, alien4cloud.ApplicationDeployed, alien4cloud.ApplicationError)
	if err != nil {
		return fmt.Errorf("failed to reach state 'Deployed' for application %q: %w", appName, err)
	}
	if state == alien4cloud.ApplicationError {
		return fmt.Errorf("failed to reach state 'Deployed' for application %q", appName)
	}
	return nil
}

func (c *suiteContext) theWorkflowWithNameIsAgainExecutable(workflowName string) error {
	if c.applicationID == "" || c.environmentID == "" {
		return fmt.Errorf("Missing mandatory context parameter applicationID: %q or environmentID: %q", c.applicationID, c.environmentID)
	}

	_, err := c.a4cClient.DeploymentService().RunWorkflow(c.ctx, c.applicationID, c.environmentID, workflowName, 10*time.Minute)
	return err
}

func (c *suiteContext) theWorkflowWithNameIsNoLongerExecutable(workflowName string) error {
	if c.applicationID == "" || c.environmentID == "" {
		return fmt.Errorf("Missing mandatory context parameter applicationID: %q or environmentID: %q", c.applicationID, c.environmentID)
	}

	_, err := c.a4cClient.DeploymentService().RunWorkflow(c.ctx, c.applicationID, c.environmentID, workflowName, 10*time.Minute)
	if err == nil {
		return fmt.Errorf("Expecting an error when run workflow %q", workflowName)
	}
	return nil
}

func (c *suiteContext) iHaveDeletedTheWorkflowNamedInTheApplication(workflowName, applicationID string) error {
	return c.createOrDeleteWorkflow(workflowName, applicationID, false)
}

func (c *suiteContext) iCreateTheWorkflowWithNameInTheApplication(workflowName, applicationID string) error {
	return c.createOrDeleteWorkflow(workflowName, applicationID, true)
}

func (c *suiteContext) createOrDeleteWorkflow(workflowName, applicationID string, create bool) error {
	environmentID, err := c.a4cClient.ApplicationService().GetEnvironmentIDbyName(c.ctx, applicationID, alien4cloud.DefaultEnvironmentName)
	if err != nil {
		return fmt.Errorf("failed to get default environmentID for application %q: %w", applicationID, err)
	}
	a4cCtx := &alien4cloud.TopologyEditorContext{
		AppID: applicationID,
		EnvID: environmentID,
	}

	if create {
		err = c.a4cClient.TopologyService().CreateWorkflow(c.ctx, a4cCtx, workflowName)
	} else {
		err = c.a4cClient.TopologyService().DeleteWorkflow(c.ctx, a4cCtx, workflowName)
	}

	if err != nil {
		return fmt.Errorf("failed to create/delete workflow %q for application %q: %w", workflowName, applicationID, err)
	}

	return c.a4cClient.TopologyService().SaveA4CTopology(c.ctx, a4cCtx)
}

func (c *suiteContext) iAddACalloperationActivityWithOperationOnTargetToTheWorkflowWithNameInTheApplication(operationName, target, workflowName, applicationID string) error {
	environmentID, err := c.a4cClient.ApplicationService().GetEnvironmentIDbyName(c.ctx, applicationID, alien4cloud.DefaultEnvironmentName)
	if err != nil {
		return fmt.Errorf("failed to get default environmentID for application %q: %w", applicationID, err)
	}
	a4cCtx := &alien4cloud.TopologyEditorContext{
		AppID: applicationID,
		EnvID: environmentID,
	}

	activity := &alien4cloud.WorkflowActivity{}
	activity.OperationCall(target, "", "tosca.interfaces.node.lifecycle.Standard", "operationName")
	err = c.a4cClient.TopologyService().AddWorkflowActivity(c.ctx, a4cCtx, workflowName, activity)

	if err != nil {
		return fmt.Errorf("failed to add activity in workflow %q for application %q: %w", workflowName, applicationID, err)
	}

	return c.a4cClient.TopologyService().SaveA4CTopology(c.ctx, a4cCtx)
}

func (c *suiteContext) iUpdateTheDeploymentWithApplication(appName string) error {
	var err error
	c.applicationID = appName

	// Use a 30m max timeout for updates
	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Minute)
	defer cancel()

	c.environmentID, err = c.a4cClient.ApplicationService().GetEnvironmentIDbyName(ctx, c.applicationID, alien4cloud.DefaultEnvironmentName)
	if err != nil {
		return fmt.Errorf("failed to get environment id for application %q: %w", appName, err)
	}

	err = c.a4cClient.DeploymentService().UpdateApplication(ctx, c.applicationID, c.environmentID)
	if err != nil {
		return fmt.Errorf("Update application %q failed: %w", appName, err)
	}

	state, err := c.a4cClient.DeploymentService().WaitUntilStateIs(ctx, c.applicationID, c.environmentID, alien4cloud.ApplicationUpdated, alien4cloud.ApplicationUpdateError)
	if err != nil {
		return fmt.Errorf("Update application %q failed: %w", appName, err)
	}
	if state == alien4cloud.ApplicationUpdateError {
		return fmt.Errorf("Update application %q failed", appName)
	}
	return nil
}

func (c *suiteContext) iHaveDeletedThePolicyNamedInTheApplication(policyName, applicationID string) error {
	environmentID, err := c.a4cClient.ApplicationService().GetEnvironmentIDbyName(c.ctx, applicationID, alien4cloud.DefaultEnvironmentName)
	if err != nil {
		return fmt.Errorf("failed to get default environmentID for application %q: %w", applicationID, err)
	}
	a4cCtx := &alien4cloud.TopologyEditorContext{
		AppID: applicationID,
		EnvID: environmentID,
	}

	err = c.a4cClient.TopologyService().DeletePolicy(c.ctx, a4cCtx, policyName)

	if err != nil {
		return fmt.Errorf("failed to delete policy %q for application %q: %w", policyName, applicationID, err)
	}

	return c.a4cClient.TopologyService().SaveA4CTopology(c.ctx, a4cCtx)
}

func (c *suiteContext) iRunTheWorkflowNamed(workflowName string) error {
	if c.applicationID == "" || c.environmentID == "" {
		return fmt.Errorf("Missing mandatory context parameter applicationID: %q or environmentID: %q", c.applicationID, c.environmentID)
	}
	exec, err := c.a4cClient.DeploymentService().RunWorkflow(c.ctx, c.applicationID, c.environmentID, workflowName, 10*time.Minute)
	if err != nil {
		return err
	}
	if exec.Status != "SUCCEEDED" {
		return fmt.Errorf("workflow %q ended with status %q", workflowName, exec.Status)
	}
	return nil
}

func (c *suiteContext) theStatusOfTheInstanceOfTheNodeNamedIs(instanceName, nodeName, expectedStatus string) error {
	if c.applicationID == "" || c.environmentID == "" {
		return fmt.Errorf("Missing mandatory context parameter applicationID: %q or environmentID: %q", c.applicationID, c.environmentID)
	}
	// TODO should use instance in a4c client
	status, err := c.a4cClient.DeploymentService().GetNodeStatus(c.ctx, c.applicationID, c.environmentID, nodeName)
	if err != nil {
		return fmt.Errorf("failed to get status for node %s-%s: %w", nodeName, instanceName, err)
	}
	if status != expectedStatus {
		return fmt.Errorf("expecting status %q for node %s-%s but got %q", expectedStatus, nodeName, instanceName, status)
	}
	return nil
}
