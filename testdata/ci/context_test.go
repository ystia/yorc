package main

import (
	"context"
	"fmt"
	"time"

	"github.com/DATA-DOG/godog"
	"github.com/DATA-DOG/godog/gherkin"
	"github.com/alien4cloud/alien4cloud-go-client/v3/alien4cloud"
)

type suiteContext struct {
	config        Config
	a4cClient     alien4cloud.Client
	applicationID string
	environmentID string
	ctx           context.Context
	cancel        context.CancelFunc
}

func (c *suiteContext) reset(s interface{}) {
	c.ctx, c.cancel = context.WithTimeout(context.Background(), 120*time.Minute)
	err := c.a4cClient.Login(context.Background())
	if err != nil {
		panic(fmt.Errorf("failed to login into Alien4CLoud: %w", err))
	}
	c.applicationID = ""
	c.environmentID = ""
}

func FeatureContext(s *godog.Suite) {
	c := &suiteContext{}

	s.BeforeSuite(func() {
		initConfig()
		var err error
		c.config, err = getConfig()
		if err != nil {
			panic(fmt.Sprintf("Failed to get config: %+v", err))
		}

		var secure bool
		if c.config.Alien4Cloud.CA != "" {
			secure = true
		}
		c.a4cClient, err = alien4cloud.NewClient(c.config.Alien4Cloud.URL, c.config.Alien4Cloud.User, c.config.Alien4Cloud.Password, c.config.Alien4Cloud.CA, secure)
		if err != nil {
			panic(fmt.Sprintf("Failed to create alien4cloud client: %+v", err))
		}
	})

	s.Step(`^I have uploaded the artifact named "([^"]*)" to Alien$`, c.iHaveUploadedTheArtifactNamedToAlien)
	s.Step(`^I have created an application named "([^"]*)" based on template named "([^"]*)"$`, c.iHaveCreatedAnApplicationNamedBasedOnTemplateNamed)
	s.Step(`^I deploy the application named "([^"]*)"$`, c.iDeployTheApplicationNamed)
	s.Step(`^I wait for "([^"]*)" seconds$`, c.iWaitForSeconds)
	s.Step(`^Kibana contains a dashboard named "([^"]*)"$`, c.kibanaContainsADashboardNamed)
	s.Step(`^There is data in Kibana$`, c.thereIsDataInKibana)
	s.Step(`^I have deployed the application named "([^"]*)"$`, c.iDeployTheApplicationNamed)
	s.Step(`^I have deleted the policy named "([^"]*)" in the application "([^"]*)"$`, c.iHaveDeletedThePolicyNamedInTheApplication)
	s.Step(`^I update the deployment with application "([^"]*)"$`, c.iUpdateTheDeploymentWithApplication)
	s.Step(`^The web page with base url stored in attribute "([^"]*)" on node "([^"]*)" with path "([^"]*)" contains a title named "([^"]*)"$`, c.theWebPageWithBaseURLStoredInAttributeOnNodeWithPathContainsATitleNamed)
	s.Step(`^I have deleted the workflow named "([^"]*)" in the application "([^"]*)"$`, c.iHaveDeletedTheWorkflowNamedInTheApplication)
	s.Step(`^The workflow with name "([^"]*)" is no longer executable$`, c.theWorkflowWithNameIsNoLongerExecutable)
	s.Step(`^I create the workflow with name "([^"]*)" in the application "([^"]*)"$`, c.iCreateTheWorkflowWithNameInTheApplication)
	s.Step(`^I add a call-operation activity with operation "([^"]*)" on target "([^"]*)" to the workflow with name "([^"]*)" in the application "([^"]*)"$`, c.iAddACalloperationActivityWithOperationOnTargetToTheWorkflowWithNameInTheApplication)
	s.Step(`^The workflow with name "([^"]*)" is again executable$`, c.theWorkflowWithNameIsAgainExecutable)
	s.Step(`^I have deployed an application named "([^"]*)" on environment named "([^"]*)"$`, c.iHaveDeployedAnApplicationNamedOnEnvironmentNamed)
	s.Step(`^I run the workflow named "([^"]*)"$`, c.iRunTheWorkflowNamed)
	s.Step(`^I run asynchronously the workflow named "([^"]*)"$`, c.iRunAsyncTheWorkflowNamed)
	s.Step(`^I cancel the last run workflow`, c.iCancelTheLastRunWorkflow)
	s.Step(`^The status of the instance "([^"]*)" of the node named "([^"]*)" is "([^"]*)"$`, c.theStatusOfTheInstanceOfTheNodeNamedIs)
	s.Step(`^The status of the workflow is finally "([^"]*)" waiting max "([^"]*)" seconds$`, c.theStatusOfTheWorkflowIsFinally)
	s.Step(`^I have built the artifact named "([^"]*)" from templates named "([^"]*)" to Alien$`, c.iHaveBuiltTheArtifactNamedFromTemplatesNamedToAlien)
	s.Step(`^The attribute "([^"]*)" of the instance "([^"]*)" of the node named "([^"]*)" is equal to the attribute "([^"]*)" of the instance "([^"]*)" of the node named "([^"]*)"$`, c.theAttributeOfTheInstanceOfTheNodeNamedIsEqualToTheAttributeOfTheInstanceOfTheNodeNamed)
	s.Step(`^I have added a policy named "([^"]*)" of type "([^"]*)" on targets "([^"]*)"$`, c.iHaveAddedAPolicyNamedOfTypeOnTargets)
	s.Step(`^The attribute "([^"]*)" of the instance "([^"]*)" of the node named "([^"]*)" is different than the attribute "([^"]*)" of the instance "([^"]*)" of the node named "([^"]*)"$`, c.theAttributeOfTheInstanceOfTheNodeNamedIsDifferentThanTheAttributeOfTheInstanceOfTheNodeNamed)
	s.Step(`^The attribute "([^"]*)" of the instance "([^"]*)" of the node named "([^"]*)" is equal to "([^"]*)"$`, c.theAttributeOfTheInstanceOfTheNodeNamedIsEqualTo)

	s.BeforeScenario(c.reset)

	s.BeforeFeature(func(f *gherkin.Feature) {
		if c.config.Infrastructure.Name != "" {
			f.Name = fmt.Sprintf("[%s] %s", c.config.Infrastructure.Name, f.Name)
		}
	})

	s.AfterScenario(func(s interface{}, scErr error) {
		if c.cancel != nil {
			defer c.cancel()
		}
		var tags []*gherkin.Tag
		switch v := s.(type) {
		case *gherkin.ScenarioOutline:
			tags = v.Tags
		case *gherkin.Scenario:
			tags = v.Tags
		}
		if tagsContains(tags, "@cleanupAlien") && (scErr == nil || !c.config.KeepFailedApplications) && c.applicationID != "" && c.environmentID != "" {
			ctx, cancel := context.WithTimeout(c.ctx, 30*time.Minute)
			defer cancel()
			c.a4cClient.DeploymentService().UndeployApplication(ctx, c.applicationID, c.environmentID)
			c.a4cClient.DeploymentService().WaitUntilStateIs(ctx, c.applicationID, c.environmentID, alien4cloud.ApplicationUndeployed)
			// Seems we need to wait a little time to avoid the terrible A4C delete deployed element error !!!
			time.Sleep(2 * time.Second)
			c.a4cClient.ApplicationService().DeleteApplication(ctx, c.applicationID)
		}

		c.a4cClient.Logout(c.ctx)
	})
}
