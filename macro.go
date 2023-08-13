package config

import (
	"fmt"
	"strings"
)

type macro struct {
	parameters map[string]string
	paramOrder []string
	properties map[string]string
}

func NewMacro(params *string) *macro {
	parameters := strings.Split(*params, string(rune(0)))
	macro := macro{
		properties: make(map[string]string),
		parameters: make(map[string]string, len(parameters)),
		paramOrder: parameters,
	}
	for _, p := range macro.paramOrder {
		macro.parameters[p] = ""
	}
	return &macro
}

func (m *macro) SetParamValues(paramValues *string, numParams int, sects *map[string]map[string]string, currSect string) error {
	parameterValues := strings.Split(*paramValues, string(rune(0)))
	// there must be same num of params and values
	if len(parameterValues) != numParams {
		// this could be a parameter less call. Check if parameter values exists as properties
		for paramName := range m.parameters {
			prop, exists := (*sects)[currSect][paramName]
			if !exists {
				return fmt.Errorf("not same num of params and values: num params = %v and num values = %v", numParams, len(m.paramOrder))
			}
			m.parameters[paramName] = prop
			// remove property, since it was not a "real property"
			delete((*sects)[currSect], paramName)
		}
		return nil
	}
	for i, p := range m.paramOrder {
		m.parameters[p] = parameterValues[i]
	}
	return nil
}
