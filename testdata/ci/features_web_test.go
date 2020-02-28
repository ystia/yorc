package main

import (
	"fmt"
	"net/http"

	"github.com/PuerkitoBio/goquery"
)

func (c *suiteContext) theWebPageWithBaseURLStoredInAttributeOnNodeWithPathContainsATitleNamed(attributeName, nodeName, urlPath, titleName string) error {
	// TODO(loicalbertin) below func should return attr for each instance
	attributes, err := c.a4cClient.DeploymentService().GetAttributesValue(c.ctx, c.applicationID, c.environmentID, nodeName, []string{attributeName})
	if err != nil {
		return fmt.Errorf("failed to get attribute %q for node %q on application %q: %w", attributeName, nodeName, c.applicationID, err)
	}
	baseURL := attributes[attributeName]

	r, err := http.DefaultClient.Get(baseURL + urlPath)
	if err != nil {
		return fmt.Errorf("failed to get url %q: %w", baseURL+urlPath, err)
	}

	document, err := goquery.NewDocumentFromResponse(r)
	if err != nil {
		return fmt.Errorf("failed to parse html from %q: %w", baseURL+urlPath, err)
	}

	var found bool
	document.Find("h1").Each(func(i int, s *goquery.Selection) {
		title := s.Text()
		if title == titleName {
			found = true
		}
	})
	if !found {
		return fmt.Errorf("title %q not found on %q", titleName, baseURL+urlPath)
	}
	return nil
}
