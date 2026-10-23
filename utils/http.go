package utils

import "strings"

func BearerToken(value string) string {
	fields := strings.Fields(value)
	if len(fields) == 2 && strings.EqualFold(fields[0], "bearer") {
		return fields[1]
	}
	return strings.TrimSpace(value)
}
