package utils

import (
	"fmt"
	"strings"
)

func GetRepoNameFromURL(url string) string {
	url = strings.TrimSuffix(url, ".git")
	parts := strings.Split(url, "/")
	if len(parts) < 2 {
		return parts[len(parts)-1]
	}
	return fmt.Sprintf("%s_%s", parts[len(parts)-2], parts[len(parts)-1])
}
