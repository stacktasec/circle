package internal

import (
	"github.com/iancoleman/strcase"
	"strings"
)

func GenRoute(suffixes []string, resource, action string) (string, string) {
	lr := strings.ToLower(resource)

	for _, s := range suffixes {
		if strings.HasSuffix(lr, s) {
			lr = strings.ReplaceAll(lr, s, "")
			break
		}
	}

	return strcase.ToSnake(lr), strcase.ToSnake(action)
}
