package metricsutil

import (
	"strings"
)

// CleanupMetricKey replaces any reserved characters in statsd/statsite and prometheus by '-'.
func CleanupMetricKey(key []string) []string {
	res := make([]string, len(key))
	for i, keyPart := range key {
		// . is the statsd separator, _ is the prometheus separator, / is not allowed by prometheus | and : are reserved separators for statsd
		res[i] = strings.Replace(keyPart, "/", "-", -1)
		res[i] = strings.Replace(res[i], ".", "-", -1)
		res[i] = strings.Replace(res[i], "_", "-", -1)
		res[i] = strings.Replace(res[i], "|", "-", -1)
		res[i] = strings.Replace(res[i], ":", "-", -1)
	}
	return res
}
