package main

import "github.com/DATA-DOG/godog/gherkin"

func tagsContains(tags []*gherkin.Tag, tagName string) bool {
	for _, t := range tags {
		if t.Name == tagName {
			return true
		}
	}
	return false
}
