package binary

import (
	"runtime"
	"strings"
	"os"
	"path/filepath"
	"fmt"

	"gopkg.in/yaml.v3"
)

type Binary struct {
	OriginalName string            `yaml:"originalName"`
	Url          string            `yaml:"url"`
	Version      string            `yaml:"version"`
	Source       []string          `yaml:"source"`
	Header       []string          `yaml:"header"`
	UrlPostfix   map[string]string `yaml:"urlPostfix"`
	SubPath      string            `yaml:"subPath"`
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
	bmanPath, err := GetBmanPath()
	if err != nil {
		// Handle the error appropriately
		return s
	}
	s = strings.ReplaceAll(s, "${binman-path}", bmanPath)
	return s
}

func (b *Binary) GetUrl() string {
	url := ReplacePlaceholders(b.Url, b.Version)
	if b.UrlPostfix != nil {
		system := runtime.GOOS
		cpu := runtime.GOARCH
		key := system + "-" + cpu
		if postfix, exists := b.UrlPostfix[key]; exists {
			url = url + postfix
		}
	}
	return url
}

func GetBmanPath() (string, error) {
	bmanPath := os.Getenv("BMAN_PATH")
	if bmanPath == "" {
		ex, err := os.Executable()
		if err != nil {
			return "", fmt.Errorf("error getting executable path: %w", err)
		}
		bmanPath = filepath.Dir(filepath.Dir(ex)) // One folder up from the executable
	}
	return bmanPath, nil
}
