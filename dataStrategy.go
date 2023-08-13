package config

import (
	"fmt"

	confContext "github.com/grufgran/config/context"
)

type dataStrategy interface {
	execute(*confContext.Context, *Config, unMarshaller, *Logger) error
}

// rowHandler for doing nothing. Handles rowTypes: comment, empty and unknown
type doNothingStrategy struct{}

// handle strings of type: comment, empty and unknown
func (dns *doNothingStrategy) execute(ctx *confContext.Context, conf *Config, um unMarshaller, logger *Logger) error {
	return nil
}

// rowHandler for sections that doesn't satisfy given claims
type skipSectionStrategy struct{}

func newSkipSectionStrategy() *skipSectionStrategy {
	s := skipSectionStrategy{}
	return &s
}

// handle skip section
func (s *skipSectionStrategy) execute(ctx *confContext.Context, conf *Config, um unMarshaller, logger *Logger) error {
	um.setSkipSectionMode(startSkipping)
	return nil
}

// rowHandler for handling [section]-like strings. Not include and macro sections though
type sectStrategy struct {
	currSect string
}

func newSectStrategy(currSect string) *sectStrategy {
	return &sectStrategy{
		currSect: currSect,
	}
}

// handle strings of type [section]
func (s *sectStrategy) execute(ctx *confContext.Context, conf *Config, um unMarshaller, logger *Logger) error {
	ctx.RunTime.SetCurrentSect(s.currSect)
	um.setSkipSectionMode(stopSkipping)
	// if this sectname is new, then it will be added to conf.sectNames
	if _, exists := conf.sects[s.currSect]; !exists {
		conf.sectNames = append(conf.sectNames, s.currSect)
	}
	return nil
}

// rowHandler for handling [include] like strings
type includeStrategy struct {
	fileName string
}

// create new include strategy
func newIncludeStrategy(fileName string) *includeStrategy {
	return &includeStrategy{
		fileName: fileName,
	}
}

// handle strings like include, includeIfExist, includeIfExistWithBasePath
func (i *includeStrategy) execute(ctx *confContext.Context, conf *Config, um unMarshaller, logger *Logger) error {
	err := readConfigFile(ctx, i.fileName, conf, logger)
	return err
}

// rowHandler for macro define sections
type macroUseStrategy struct {
	macroName      string
	macroParams    string
	numMacroParams int
}

func newMacroUseStrategy(name, params string, numParams int) *macroUseStrategy {
	return &macroUseStrategy{
		macroName:      name,
		macroParams:    params,
		numMacroParams: numParams,
	}
}

// handle strings of type [section]
func (mus *macroUseStrategy) execute(ctx *confContext.Context, conf *Config, um unMarshaller, logger *Logger) error {
	// find macro
	if macro, exists := conf.macros[mus.macroName]; !exists {
		return fmt.Errorf("macro %v not found", mus.macroName)
		// set param values
	} else if err := macro.SetParamValues(&mus.macroParams, mus.numMacroParams, &conf.sects, ctx.RunTime.Params[confContext.CurrSect]); err != nil {
		return err
		// Add macro props to sect
	} else if ctx.RunTime.SaveTo == confContext.Sects {
		if err := conf.addMacroPropsToSect(ctx, &mus.macroName); err != nil {
			return err
		}
		// Add macro props to macro
	} else {
		if err := conf.addPropsToMacro(ctx, &mus.macroName); err != nil {
			return err
		}
	}
	return nil
}

// rowHandler for macro define sections
type macroDefineStrategy struct {
	macroName   string
	macroParams string
}

func newMacroDefineStrategy(name, params string) *macroDefineStrategy {
	return &macroDefineStrategy{
		macroName:   name,
		macroParams: params,
	}
}

// handle strings of type [section]
func (m *macroDefineStrategy) execute(ctx *confContext.Context, conf *Config, um unMarshaller, logger *Logger) error {
	ctx.RunTime.SetCurrentMacro(m.macroName)
	conf.macros[m.macroName] = NewMacro(&m.macroParams)
	um.setSkipSectionMode(stopSkipping)
	return nil
}

// rowHandler for handling key=value like strings
type propertyStrategy struct {
	rowType dataType
	key     string
	value   string
}

// create new propertyHandler
func newPropertyStrategy(rowType dataType, key string, value string) *propertyStrategy {
	return &propertyStrategy{
		rowType: rowType,
		key:     key,
		value:   value,
	}
}

// handle strings of type: property and multiline
func (ps *propertyStrategy) execute(ctx *confContext.Context, conf *Config, um unMarshaller, logger *Logger) error {
	// if it is a multiLineHereDoc rowType then the value contains the hereDocMarker
	if ps.rowType == multiLineHereDoc {
		hereDocMarker := ps.value

		// Loop until we finds the other hereDoc
		for {
			if um.scan() {
				if err := um.prepareData(ctx, conf); err != nil {
					return err
				}
				data := um.getFileRowData()

				// return if we got the hereDoc marker
				if data.value == hereDocMarker {
					break
				}
				// add data to property
				conf.appendProperty(ctx, ps.key, data.value, "\n")

				// keep rowType = multiLineHereDoc, until we find the hereDocMarker
				data.rowType = multiLineHereDoc
			} else {
				// no more rows in file. Time to leave
				break
			}
		}
		return nil
	}

	// add/update property to current sect
	conf.upsertProperty(ctx, ps.key, ps.value)

	// handle multiLineBackslash
	if ps.rowType == multiLineBackslash {
		// Loop until we got a row without ending backslash
		for {
			if um.scan() {
				if err := um.prepareData(ctx, conf); err != nil {
					return err
				}
				data := um.getFileRowData()

				// add data to property
				conf.appendProperty(ctx, ps.key, data.value)

				// break when there is no ending backslash
				if data.rowType != multiLineBackslash {
					break
				}
			} else {
				// no more rows in file. Time to leave
				break
			}
		}
	}
	return nil
}
