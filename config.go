package config

import (
	"fmt"
	"strings"

	confContext "github.com/grufgran/config/context"

	"github.com/grufgran/config/stringMask"
)

type Config struct {
	sectNames []string
	sects     map[string]map[string]string
	macros    map[string]*macro
	filesUsed []string
}

func NewConfig() *Config {
	conf := &Config{
		sects: make(map[string]map[string]string),
	}
	return conf
}

func NewConfigFromFile(ctx *confContext.Context, fileName string, logger *Logger) (*Config, error) {

	// create ctx if not provided
	if ctx == nil {
		ctx = confContext.NewContext(nil)
		ctx.SetConfRoot(fileName)
	} else {
		if _, err := ctx.GetConfRoot(); err != nil {
			ctx.SetConfRoot(fileName)
		}
	}

	// create conf
	conf := NewConfig()

	// read file
	err := readConfigFile(ctx, fileName, conf, logger)
	if err != nil {
		return conf, err
	}

	// return
	return conf, nil
}

func (conf *Config) SectNames() []string {
	return conf.sectNames
}

func (conf *Config) Sect(name string) *Sect {
	_, exists := conf.sects[name]
	sect := newSect(name, exists, conf)
	return sect
}

func (conf *Config) PropVal(sectName string, propName string) (string, bool) {
	if sect, exists := conf.sects[sectName]; !exists {
		return "", exists
	} else {
		propVal, exists := sect[propName]
		return propVal, exists
	}
}

func (conf *Config) PropOrDefault(sectName string, propName string, defVal string) string {
	if val, exists := conf.PropVal(sectName, propName); exists {
		return val
	}
	return defVal
}

// Provide absolutePath for conf file
func (conf *Config) ConfFileName() string {
	return conf.filesUsed[0]
}

type Logger interface {
	Debug(args ...any)
	Debugf(format string, args ...any)
}

// Replace all constants on row with config values
func (conf *Config) replaceConstants(sm *stringMask.StringMask, currentMask, setMaskTo rune) error {
	// Get squere brackets
	leftSquereBrackets, rightSquereBrackets, err := sm.GetMaskPointsForOppositeRunes('[', ']', currentMask)
	if err != nil {
		return err
	}

	// no squere brackets, no show
	if len(*leftSquereBrackets) == 0 {
		return nil
	}
	// get all colons
	colons := sm.GetAllMaskPoints(':', currentMask)

	// no colons, no show
	if len(*colons) > 0 {
		for {
			// find most narrow bracket pair. Since brackets could be nested, like this [[s:pp]],
			// we will start exemining the most narrow bracket space
			minWidth := -1
			minIndex := -1
			for i := range *leftSquereBrackets {
				width := (*rightSquereBrackets)[i].Pos - (*leftSquereBrackets)[i].Pos
				if width < minWidth || minWidth == -1 {
					minIndex = i
					minWidth = width
				}
			}
			// check if there is a colon between the brackets
			for i := range *colons {
				if (*colons)[i].Pos > (*leftSquereBrackets)[minIndex].Pos && (*colons)[i].Pos < (*rightSquereBrackets)[minIndex].Pos {
					// we have found a constant!
					sect := sm.GetStringBetween((*leftSquereBrackets)[minIndex].Pos+1, (*colons)[i].Pos-1, true)
					prop := sm.GetStringBetween((*colons)[i].Pos+1, (*rightSquereBrackets)[minIndex].Pos-1, true)
					if val, exists := conf.sects[sect][prop]; exists {
						// incredible, it was found!
						sm.MaskBetween((*leftSquereBrackets)[minIndex].Pos, (*rightSquereBrackets)[minIndex].Pos, setMaskTo)
						sm.NewTagAtPos((*leftSquereBrackets)[minIndex].Pos, val)
						break
					} else {
						constant := sm.GetStringBetween((*leftSquereBrackets)[minIndex].Pos, (*colons)[i].Pos, false)
						return fmt.Errorf("could not replace constant %v in %v. Constant value not found", constant, string(sm.String))
					}
				}
			}
			// remove processed brackets. If this was the last one, there is no need to remove it from the slice. Just break the loop
			if len(*leftSquereBrackets) == 1 {
				break
			}
			leftSquereBrackets = stringMask.RemoveMaskPointFromSlice(leftSquereBrackets, minIndex)
			rightSquereBrackets = stringMask.RemoveMaskPointFromSlice(rightSquereBrackets, minIndex)
		}
	}
	return nil
}

func (conf *Config) hasRequiredClaims(sm *stringMask.StringMask, currentMask, setMaskTo rune, ctx *confContext.Context) (bool, error) {
	// ok, so we have a section, and now its time to see if it has any claims.
	// if there is a question mark on the row, then there are claims
	questionMark := sm.GetFirstRunePoint('?', currentMask)
	if questionMark == nil || questionMark.Pos < 2 {
		// no question mark found, then we can exit happily, saying claims exits and no error
		return true, nil
	} else {
		// mask the questionMark
		sm.MaskAtPos(questionMark.Pos, '?')
		sm.MaskRightSpacesFromPos(questionMark.Pos+1, 'X', currentMask)
	}
	// get pos for left squere bracket
	sectionStart := sm.GetFirstRunePoint('[', '[')

	// mask the claims area. mask with 'c'
	sm.MaskBetween(sectionStart.Pos+1, questionMark.Pos-1, setMaskTo)

	// clean out white spaces if there are any
	sm.MaskRightSpacesFromPos(sectionStart.Pos+1, 'X', setMaskTo)
	sm.MaskLeftSpacesFromPos(questionMark.Pos-1, 'X', setMaskTo)

	// mask "," claim delimiters
	commas := sm.Mask(',', 'd', setMaskTo)

	// mask "&" claim delimiters
	ands := sm.Mask('&', 'd', setMaskTo)

	// There can not be both commas and ands
	if len(*commas) > 0 && len(*ands) > 0 {
		return false, fmt.Errorf("there can not be both \",\" (commas) and \"&\" in claims sektion: %s", string(sm.String))
	}
	// Mask whitespace around delimiters
	sm.MaskLeftRightSpacesAroundPoints(commas, 'X', setMaskTo)
	sm.MaskLeftRightSpacesAroundPoints(ands, 'X', setMaskTo)

	// get all claims
	claims := sm.GetStrings(setMaskTo)

	// Loop claims and replace constants
	for i := range claims {
		sect, prop, isConstant := conf.isConstant(&claims[i])
		if isConstant {
			if val, exists := conf.sects[sect][prop]; exists {
				claims[i] = val
			} else {
				return false, fmt.Errorf("could not replace constant %v in %v. Constant value not found", claims[i], string(sm.String))
			}
		}
	}

	// Loop claims and see if they apply
	// [claim1&claim2&claim3?...] all claims must match
	// [claim1,claim2,claim3?...] any claim must match
	for i := range claims {
		if ctx.Claims.Has(claims[i]) {
			if len(*commas) > 0 {
				return true, nil
			}
		} else if len(*commas) == 0 {
			return false, nil
		}
	}
	return true, nil
}

// Check if this is a constant. Constant example: "[some sect:some property]"
func (conf *Config) isConstant(s *string) (string, string, bool) {

	// To start with, a constant must start and end with []
	if strings.HasPrefix(*s, "[") && strings.HasSuffix(*s, "]") {
		// is there a colon present?
		if colon := strings.Index(*s, ":"); colon != -1 {
			// extract sect and prop
			sect := strings.TrimSpace((*s)[1:colon])
			prop := strings.TrimSpace((*s)[colon+1 : len(*s)-1])
			return sect, prop, true
		}
	}
	return "", "", false
}

func (c *Config) upsertProperty(ctx *confContext.Context, key, value string) {
	if ctx.RunTime.SaveTo == confContext.Sects {
		// Get current section
		cs := ctx.RunTime.Params[confContext.CurrSect]
		if prop, exists := c.sects[cs]; exists {
			prop[key] = value
			c.sects[cs] = prop
		} else {
			c.sects[cs] = map[string]string{key: value}
		}
	} else {
		// Get current macro
		cm := ctx.RunTime.Params[confContext.CurrMacro]
		// always append to macro body
		m := c.macros[cm]
		m.properties[key] = value
	}
}

func (c *Config) appendProperty(ctx *confContext.Context, key string, values ...string) {
	if ctx.RunTime.SaveTo == confContext.Sects {
		// Get current section
		cs := ctx.RunTime.Params[confContext.CurrSect]
		if prop, exists := c.sects[cs][key]; exists {
			var sb strings.Builder
			sbSize := len(prop) + c.getValuesLen(values...)
			sb.Grow(sbSize)
			sb.WriteString(prop)
			// separate props with \n if there is more than one value
			if len(values) > 1 {
				sb.WriteString("\n")
			}
			sb.WriteString(values[0])
			c.sects[cs][key] = sb.String()
		} else {
			c.sects[cs] = map[string]string{key: values[0]}
		}
	} else {
		// Get current macro
		macro := c.getCurrentMacro(ctx)
		// always append to macro properties
		if prop, exists := macro.properties[key]; exists {
			var sb strings.Builder
			sbSize := len(prop) + c.getValuesLen(values...)
			sb.Grow(sbSize)
			sb.WriteString(prop)
			// separate props with \n if there is more than one value
			if len(values) > 1 {
				sb.WriteString("\n")
			}
			sb.WriteString(values[0])
			macro.properties[key] = sb.String()
		} else {
			macro.properties[key] = values[0]
		}
	}
}

func (conf *Config) getValuesLen(values ...string) int {
	l := 0
	for _, v := range values {
		l += len(v)
	}
	return l
}

func (conf *Config) addMacroPropsToSect(ctx *confContext.Context, macroName *string) error {

	// Get current section
	cs := ctx.RunTime.Params[confContext.CurrSect]
	macro := conf.macros[*macroName]

	// get config sectProps
	sectProps, sectExists := conf.sects[cs]
	if !sectExists {
		sectProps = make(map[string]string)
	}
	for k, v := range macro.properties {
		if v, err := conf.applyParamsAndConstants(v, &macro.parameters); err != nil {
			return err
		} else {
			sectProps[k] = v
		}
	}
	conf.sects[cs] = sectProps
	return nil
}

func (conf *Config) addPropsToMacro(ctx *confContext.Context, macroName *string) error {

	// Get current macro
	cm := ctx.RunTime.Params[confContext.CurrMacro]
	currMacro := conf.macros[cm]
	useMacro := conf.macros[*macroName]

	// add useMacro props to currMacro
	for k, v := range useMacro.properties {
		if v, err := conf.applyParamsAndConstants(v, &useMacro.parameters); err != nil {
			return err
		} else {
			currMacro.properties[k] = v
		}
	}
	return nil
}

func (conf *Config) applyParamsAndConstants(v string, params *map[string]string) (string, error) {
	// create new stringmask on v
	sm := stringMask.NewStringMask(v, '-')
	// begin with replacing all params with real values, ex {$p1} => "val1"
	// first find all curly brackets
	if cbs, cbe, err := sm.GetMaskPointsForOppositeRunes('{', '}', '-'); err != nil {
		return "", err
	} else {
		// loop thru all curly brackets
		for i, cb := range *cbs {
			// check if there is a parameter within brackets
			pc := sm.GetStringBetween(cb.Pos+1, (*cbe)[i].Pos-1, true)
			if pv, exists := (*params)[pc]; exists {
				// ok, we found a parameter. Mask this and set tag
				sm.MaskBetween(cb.Pos, (*cbe)[i].Pos, 'p')
				sm.NewTagAtPos(cb.Pos, pv)
			}
		}
	}
	// replace all constants
	if err := conf.replaceConstants(sm, '-', 'p'); err != nil {
		return "", err
	}
	return sm.GetString('-', 'p'), nil
}

func (conf *Config) getCurrentMacro(ctx *confContext.Context) *macro {
	cm := ctx.RunTime.Params[confContext.CurrMacro]
	m := conf.macros[cm]
	return m
}
