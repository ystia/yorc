package main

import (
	"fmt"
	"strings"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
)

func (c *suiteContext) getKibanaURL() (string, error) {
	attrs, err := c.a4cClient.DeploymentService().GetAttributesValue(c.ctx, c.applicationID, c.environmentID, "Kibana", []string{"url"})
	if err != nil {
		return "", err
	}
	// returns http://34.76.175.45:5601/app/kibana#/dashboards
	s := strings.Split(attrs["url"], "/")

	return s[0] + "//" + s[2], nil
}

func (c *suiteContext) kibanaContainsADashboardNamed(dashboardName string) error {
	url, err := c.getKibanaURL()
	if err != nil {
		return err
	}
	ctx, cancel := c.getChromeDPContext()
	defer cancel()

	sel := fmt.Sprintf(`//a[text()[contains(., '%s')]]`, dashboardName)
	return chromedp.Run(ctx,
		chromedp.Navigate(url+"/app/kibana#/dashboards"),
		chromedp.WaitVisible(sel),
	)
}

func (c *suiteContext) thereIsDataInKibana() error {
	url, err := c.getKibanaURL()
	if err != nil {
		return err
	}
	ctx, cancel := c.getChromeDPContext()
	defer cancel()
	var nodes []*cdp.Node
	err = chromedp.Run(ctx,
		chromedp.Navigate(url+"/app/kibana#/discover"),
		chromedp.Nodes(".discover-table-sourcefield", &nodes),
	)
	if err != nil {
		return err
	}
	if len(nodes) == 0 {
		return fmt.Errorf("Kibana discover panel does not contain data")
	}
	return nil
}
