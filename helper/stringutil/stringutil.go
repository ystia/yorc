package stringutil

import (
	"strconv"
	"strings"
	"time"
)

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

// UniqueTimestampedName generates a time-stamped name for temporary file or directory by instance
func UniqueTimestampedName(prefix string, suffix string) string {
	return prefix + strconv.FormatInt(time.Now().UnixNano(), 10) + suffix
}
