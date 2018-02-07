package labelsutil

import "novaforge.bull.com/starlings-janus/janus/helper/labelsutil/internal"

// Matches checks if a given string representation of a filter matches a set of labels
func Matches(labels map[string]string, filter string) (bool, error) {
	f, err := internal.FilterFromString(filter)
	if err != nil {
		return false, err
	}
	return f.Matches(labels)
}

// MatchesAll checks if all of the given filters match a set of labels
//
// Providing no filters is not considered as an error and will return true.
func MatchesAll(labels map[string]string, filters ...string) (bool, error) {
	for _, filter := range filters {
		m, err := Matches(labels, filter)
		if err != nil || !m {
			return m, err
		}
	}
	return true, nil
}
