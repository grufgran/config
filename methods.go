package config

import "fmt"

// Return a slice of all sectname
func (conf *Config) GetAllSectNames() []string {

	sectNames := make([]string, len(conf.sects))
	i := 0
	for key := range conf.sects {
		sectNames[i] = key
		i++
	}
	return sectNames
}

func (conf *Config) GetProperties(sect string) (map[string]string, error) {
	props, exists := conf.sects[sect]
	if !exists {
		return nil, fmt.Errorf("section %s not found", sect)
	}
	return props, nil
}
