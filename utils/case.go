package utils

import (
	"strings"

	"github.com/iancoleman/strcase"
)

func CamelCase(s string) string {
	return strcase.ToCamel(s)
}

func LowerCamel(s string) string {
	return strcase.ToLowerCamel(s)
}

func Lower(s string) string {
	return strings.ToLower(s)
}
