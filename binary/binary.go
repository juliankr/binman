package binary

import (
	"gopkg.in/yaml.v3"
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
