package stringutil

import "strings"

// GetLastElement returns the last element of a "separator-separated" string
func GetLastElement(str string, separator string) string {
	var ret string
	if strings.Contains(str, separator) {
		idx := strings.LastIndex(str, separator)
		ret = str[idx+1:]
	}
	return ret
}

// GetAllExceptLastElement returns all elements except the last one of a "separator-separated" string
func GetAllExceptLastElement(str string, separator string) string {
	var ret string
	if strings.Contains(str, separator) {
		idx := strings.LastIndex(str, separator)
		ret = str[:idx]
	}
	return ret
}
