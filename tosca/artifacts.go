package tosca

type ArtifactDefinition struct {
	Type        string `yaml:"type"`
	File        string `yaml:"file"`
	Description string `yaml:"description,omitempty"`
	Repository  string `yaml:"repository,omitempty"`
	DeployPath  string `yaml:"deploy_path,omitempty"`
}

func (a *ArtifactDefinition) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err == nil {
		a.File = s
		return nil
	}
	var str struct {
		Type        string `yaml:"type"`
		File        string `yaml:"file"`
		Description string `yaml:"description,omitempty"`
		Repository  string `yaml:"repository,omitempty"`
		DeployPath  string `yaml:"deploy_path,omitempty"`
	}
	if err := unmarshal(&str); err != nil {
		return err
	}
	a.Type = str.Type
	a.File = str.File
	a.Description = str.Description
	a.Repository = str.Repository
	a.DeployPath = str.DeployPath
	return nil
}
