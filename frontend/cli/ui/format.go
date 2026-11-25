package ui

import "strings"

func friendlyID(id string) string {
	id = strings.TrimSpace(id)
	if _, value, found := strings.Cut(id, "."); found {
		id = value
	}

	return strings.ReplaceAll(id, "_", " ")
}
