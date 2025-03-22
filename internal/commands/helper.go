package commands

import (
	"strings"
)

func Int(i int) *int {
	return &i
}

func String(s string) *string {
	return &s
}

func customId(components ...string) string {
	return strings.Join(components, "#")
}
