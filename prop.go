package config

import (
	"fmt"
	"strconv"
)

type Prop struct {
	name   string
	value  string
	exists bool
}

func newProp(name string, value string, exists bool) *Prop {
	prop := Prop{
		name:   name,
		value:  value,
		exists: exists,
	}
	return &prop
}

func (p *Prop) Value() (string, error) {
	if !p.exists {
		err := fmt.Errorf("property %v does not exists", p.name)
		return "", err
	}
	return p.value, nil
}

func (p *Prop) ValueOrDefault(def any) (any, error) {
	if !p.exists {
		return def, nil
	}
	propValue := p.value

	if _, isString := def.(string); isString {
		return propValue, nil
	}

	if _, isInt := def.(int); isInt {
		val, err := strconv.Atoi(propValue)
		return val, err
	}

	err := fmt.Errorf("datatype not implemented")
	return def, err
}
