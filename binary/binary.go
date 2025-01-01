package binary

import (
	"runtime"
	"strings"

	"gopkg.in/yaml.v3"
)

type Binary struct {
	OriginalName string            `yaml:"originalName"`
	Url          string            `yaml:"url"`
	Version      string            `yaml:"version"`
	Source       []string          `yaml:"source"`
	Header       []string          `yaml:"header"`
	UrlPostfix   map[string]string `yaml:"urlPostfix"`
}

func (b *Binary) ToYAML() (string, error) {
	data, err := yaml.Marshal(b)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func ReplacePlaceholders(s, version, binmanPath string) string {
	system := runtime.GOOS
	cpu := runtime.GOARCH

	s = strings.ReplaceAll(s, "${version}", version)
	s = strings.ReplaceAll(s, "${system}", system)
	s = strings.ReplaceAll(s, "${cpu}", cpu)
	s = strings.ReplaceAll(s, "${binman-path}", binmanPath)
	return s
}

func (b *Binary) GetUrl() string {
	url := ReplacePlaceholders(b.Url, b.Version, "")
	if b.UrlPostfix != nil {
		system := runtime.GOOS
		cpu := runtime.GOARCH
		key := system + "-" + cpu
		if postfix, exists := b.UrlPostfix[key]; exists {
			url = url + "/" + postfix
		}
	}
	return url
}
