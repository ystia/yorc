package main

import (
	"fmt"
	"strings"
	"time"
)

func (c *suiteContext) iWaitForSeconds(wait string) error {
	waitDuration, err := time.ParseDuration(strings.TrimSpace(wait) + "s")
	if err != nil {
		return fmt.Errorf("invalid argument %q for iWaitForSeconds step: %w", wait, err)
	}
	time.Sleep(waitDuration)
	return nil
}

func (c *suiteContext) iHaveDeployedAnApplicationNamedOnEnvironmentNamed(applicationName, environmentName string) error {
	c.applicationID = applicationName
	var err error
	c.environmentID, err = c.a4cClient.ApplicationService().GetEnvironmentIDbyName(c.ctx, c.applicationID, environmentName)
	return err
}
