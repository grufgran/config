package config

type Sect struct {
	name   string
	Exists bool
	conf   *Config
}

func newSect(name string, exists bool, conf *Config) *Sect {
	sect := &Sect{
		name:   name,
		Exists: exists,
		conf:   conf}
	return sect
}

// Get index number for sect
func (sect *Sect) Index() int {
	if !sect.Exists {
		return -1
	}
	for i, name := range sect.conf.sectNames {
		if sect.name == name {
			return i
		}
	}
	// Strange, we should newer end up here
	return -1
}
func (sect *Sect) PropValOrDefault(propName string, defaultValue string) string {
	if !sect.Exists {
		return defaultValue
	}
	if propVal, exists := sect.conf.sects[sect.name][propName]; exists {
		return propVal
	} else {
		return defaultValue
	}
}

func (sect *Sect) PropVal(propName string) (string, bool) {
	if !sect.Exists {
		return "", false
	}
	propVal, exists := sect.conf.sects[sect.name][propName]
	return propVal, exists
}

func (sect *Sect) Prop(name string) *Prop {
	if !sect.Exists {
		return newProp(name, "", false)
	}
	val, exists := sect.conf.sects[sect.name][name]
	return newProp(name, val, exists)
}
