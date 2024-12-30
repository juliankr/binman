package binary

import (
	"gopkg.in/yaml.v3"
	"runtime"
	"strings"
)

type Binary struct {
	OriginalName string
	Url          string
	Version      string
}

func (b *Binary) ToYAML() (string, error) {
	data, err := yaml.Marshal(b)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func ReplacePlaceholders(s, version string) string {
	system := runtime.GOOS
	cpu := runtime.GOARCH

	s = strings.ReplaceAll(s, "${version}", version)
	s = strings.ReplaceAll(s, "${system}", system)
	s = strings.ReplaceAll(s, "${cpu}", cpu)
	return s
}
