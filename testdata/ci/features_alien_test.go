package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/alien4cloud/alien4cloud-go-client/v3/alien4cloud"
)

const componentsDir = "components"

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

func (c *suiteContext) iRunAsyncTheWorkflowNamed(workflowName string) error {
	if c.applicationID == "" || c.environmentID == "" {
		return fmt.Errorf("Missing mandatory context parameter applicationID: %q or environmentID: %q", c.applicationID, c.environmentID)
	}

	go func() {
		c.a4cClient.DeploymentService().RunWorkflow(c.ctx, c.applicationID, c.environmentID, workflowName, 10*time.Minute)
	}()
	return nil
}

func (c *suiteContext) iCancelTheLastRunWorkflow() error {
	if c.applicationID == "" || c.environmentID == "" {
		return fmt.Errorf("Missing mandatory context parameter applicationID: %q or environmentID: %q", c.applicationID, c.environmentID)
	}

	wfExec, err := c.a4cClient.DeploymentService().GetLastWorkflowExecution(c.ctx, c.applicationID, c.environmentID)
	if err != nil {
		return err
	}

	if wfExec == nil || wfExec.Execution.ID == "" {
		return fmt.Errorf("failed to get last workflow executionID for applicationID: %q or environmentID: %q", c.applicationID, c.environmentID)
	}
	return c.a4cClient.DeploymentService().CancelExecution(c.ctx, c.environmentID, wfExec.Execution.ID)
}

func (c *suiteContext) theStatusOfTheWorkflowIsFinally(expectedStatus string, wait string) error {
	timeout, err := time.ParseDuration(strings.TrimSpace(wait) + "s")
	if err != nil {
		return fmt.Errorf("invalid argument %q for theStatusOfTheWorkflowIsFinally step: %w", wait, err)
	}

	if c.applicationID == "" || c.environmentID == "" {
		return fmt.Errorf("Missing mandatory context parameter applicationID: %q or environmentID: %q", c.applicationID, c.environmentID)
	}

	timeAfter := time.After(timeout)

	var status string
	for status != expectedStatus {
		wfExec, err := c.a4cClient.DeploymentService().GetLastWorkflowExecution(c.ctx, c.applicationID, c.environmentID)
		if err != nil {
			return err
		}

		if wfExec == nil || wfExec.Execution.Status == "" {
			return fmt.Errorf("failed to get last workflow status for applicationID: %q or environmentID: %q", c.applicationID, c.environmentID)
		}

		status = wfExec.Execution.Status

		select {
		case <-timeAfter:
			return fmt.Errorf("unexpected workflow status %q instead of %q after timeout:%s", status, expectedStatus, timeout.String())
		default:
		}
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

func (c *suiteContext) iHaveBuiltTheArtifactNamedFromTemplatesNamedToAlien(artifactName, templateName string) error {
	// Check the zip isn't already done
	fInfo, err := os.Stat(componentsDir)
	if err != nil {
		if os.IsNotExist(err) {
			err = os.Mkdir(componentsDir, 0777)
			if err != nil {
				return fmt.Errorf("failed to create directory %q: %w", componentsDir, err)
			}
			fInfo, _ = os.Stat(componentsDir)
		} else {
			return fmt.Errorf("failed to stat directory components: %w", err)
		}
	}

	if !fInfo.IsDir() {
		return fmt.Errorf("%q must be a directory", componentsDir)
	}
	target := filepath.Join("components", artifactName+".zip")
	matches, err := filepath.Glob(target)
	if err != nil {
		return fmt.Errorf("failed to find CSARs matching %q: %w", target, err)
	}
	if len(matches) != 0 {
		// nothing to do
		return nil
	}

	source := filepath.Join("templates", templateName)
	err = zipDirectory(source, target)
	if err != nil {
		fmt.Errorf("failed to zip directory:%q due to error:%v", source, err)
	}
	return nil
}

func (c *suiteContext) getInstanceAttributeValue(nodeName, instance, attribute string) (string, error) {
	var value string
	if c.applicationID == "" || c.environmentID == "" {
		return value, fmt.Errorf("Missing mandatory context parameter applicationID: %q or environmentID: %q", c.applicationID, c.environmentID)
	}
	attrs, err := c.a4cClient.DeploymentService().GetInstanceAttributesValue(c.ctx, c.applicationID, c.environmentID, nodeName, instance, []string{attribute})
	if err != nil || attrs == nil {
		return value, err
	}
	return attrs[attribute], nil
}

func (c *suiteContext) theAttributeOfTheInstanceOfTheNodeNamedIsEqualToTheAttributeOfTheInstanceOfTheNodeNamed(attribute1, instance1, nodeName1, attribute2, instance2, nodeName2 string) error {
	return c.compareInstanceAttributeValues(attribute1, instance1, nodeName1, attribute2, instance2, nodeName2, true)
}

func (c *suiteContext) compareInstanceAttributeValues(attribute1, instance1, nodeName1, attribute2, instance2, nodeName2 string, wantEquals bool) error {
	if c.applicationID == "" || c.environmentID == "" {
		return fmt.Errorf("Missing mandatory context parameter applicationID: %q or environmentID: %q", c.applicationID, c.environmentID)
	}
	val1, err := c.getInstanceAttributeValue(nodeName1, instance1, attribute1)
	if err != nil {
		return err
	}
	val2, err := c.getInstanceAttributeValue(nodeName2, instance2, attribute2)
	if err != nil {
		return err
	}

	if !wantEquals == (val1 == val2) {
		return fmt.Errorf("value %q (attribute: %q, instance: %q, node: %q) and value %q (attribute: %q, instance: %q, node: %q) ewpected equals:%t is false", val1, attribute1, instance1, nodeName1, val2, attribute2, instance2, nodeName2, wantEquals)
	}
	return nil
}

func (c *suiteContext) iHaveAddedAPolicyNamedOfTypeOnTargets(policyName, policyType, targets string) error {
	if c.applicationID == "" || c.environmentID == "" {
		return fmt.Errorf("Missing mandatory context parameter applicationID: %q or environmentID: %q", c.applicationID, c.environmentID)
	}
	a4cCtx := &alien4cloud.TopologyEditorContext{
		AppID: c.applicationID,
		EnvID: c.environmentID,
	}

	err := c.a4cClient.TopologyService().AddPolicy(c.ctx, a4cCtx, policyName, policyType)
	if err != nil {
		return fmt.Errorf("failed to add policy with name:%q due to error:%v", policyName, err)
	}
	targetsSlice := strings.Split(targets, ",")

	err = c.a4cClient.TopologyService().AddTargetsToPolicy(c.ctx, a4cCtx, policyName, targetsSlice)
	if err != nil {
		return fmt.Errorf("failed to add targets:%q to policy with name:%q due to error:%v", targetsSlice, policyName, err)
	}
	return c.a4cClient.TopologyService().SaveA4CTopology(c.ctx, a4cCtx)
}

func (c *suiteContext) theAttributeOfTheInstanceOfTheNodeNamedIsDifferentThanTheAttributeOfTheInstanceOfTheNodeNamed(attribute1, instance1, nodeName1, attribute2, instance2, nodeName2 string) error {
	return c.compareInstanceAttributeValues(attribute1, instance1, nodeName1, attribute2, instance2, nodeName2, false)
}

func (c *suiteContext) theAttributeOfTheInstanceOfTheNodeNamedIsEqualTo(attribute, instance, nodeName, expected string) error {
	if c.applicationID == "" || c.environmentID == "" {
		return fmt.Errorf("Missing mandatory context parameter applicationID: %q or environmentID: %q", c.applicationID, c.environmentID)
	}
	val, err := c.getInstanceAttributeValue(nodeName, instance, attribute)
	if err != nil {
		return err
	}

	if val != expected {
		return fmt.Errorf("value %q (attribute: %q, instance: %q, node: %q) is not equal to expected: %s", val, attribute, instance, nodeName, expected)
	}
	return nil
}
