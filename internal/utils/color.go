package utils

import "strings"

func FormatColor(color string) string {
	r := strings.NewReplacer(
		"\\033", "\x1b",
		"\033", "\x1b",
		"\\e", "\x1b",
	)
	return r.Replace(color)
}
