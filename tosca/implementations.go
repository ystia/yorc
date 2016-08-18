package tosca

type Implementation struct {
	Primary      string   `yaml:"primary"`
	Dependencies []string `yaml:"dependencies,omitempty"`
}

func (i *Implementation) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err == nil {
		i.Primary = s
		return nil
	}
	var str struct {
		Primary      string   `yaml:"primary"`
		Dependencies []string `yaml:"dependencies,omitempty"`
	}
	if err := unmarshal(&str); err != nil {
		return err
	}
	i.Primary = str.Primary
	i.Dependencies = str.Dependencies
	return nil
}
