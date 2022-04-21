package tools

import (
	"fmt"
	"unicode/utf8"
)

func Contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}
	return false
}
