package config

import (
	config "config/context"
	"config/stringMask"
	"fmt"
	"strings"
)

type Config struct {
	Sects map[string]map[string]string
}

type Logger interface {
	Debug(args ...interface{})
	Debugf(format string, args ...interface{})
}

func NewConfig() *Config {
	conf := &Config{
		Sects: make(map[string]map[string]string),
	}
	return conf
}

func NewConfigFromFile(ctx *config.Context, fileName string, logger *Logger) (*Config, error) {

	// create ctx if not provided
	if ctx == nil {
		ctx = config.NewContext(nil)
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

// Replace all constants on row with config values
func replaceConstants(sm *stringMask.StringMask, currentMask, setMaskTo rune, conf *Config) error {
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
					if val, exists := conf.Sects[sect][prop]; exists {
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

func hasRequiredClaims(sm *stringMask.StringMask, currentMask, setMaskTo rune, conf *Config, ctx *config.Context) (bool, error) {
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
		sect, prop, isConstant := isConstant(&claims[i])
		if isConstant {
			if val, exists := conf.Sects[sect][prop]; exists {
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
func isConstant(s *string) (string, string, bool) {

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

func upsertProperty(ctx *config.Context, c *Config, key, value string) {
	if ctx.RunTime.SaveTo == config.Sects {
		// Get current section
		cs := ctx.RunTime.Params[config.CurrSect]
		if prop, exists := c.Sects[cs]; exists {
			prop[key] = value
			c.Sects[cs] = prop
		} else {
			c.Sects[cs] = map[string]string{key: value}
		}
	} else {
		// Get current macro
		cm := ctx.RunTime.Params[config.CurrMacro]
		// always append to macro body
		m := ctx.RunTime.Macros[cm]
		m.Properties[key] = value
	}
}

func appendProperty(ctx *config.Context, c *Config, key string, values ...string) {
	if ctx.RunTime.SaveTo == config.Sects {
		// Get current section
		cs := ctx.RunTime.Params[config.CurrSect]
		if prop, exists := c.Sects[cs][key]; exists {
			var sb strings.Builder
			sbSize := len(prop) + getValuesLen(values...)
			sb.Grow(sbSize)
			sb.WriteString(prop)
			// separate props with \n if there is more than one value
			if len(values) > 1 {
				sb.WriteString("\n")
			}
			sb.WriteString(values[0])
			c.Sects[cs][key] = sb.String()
		} else {
			c.Sects[cs] = map[string]string{key: values[0]}
		}
	} else {
		// Get current macro
		macro := ctx.RunTime.GetCurrentMacro()
		// always append to macro properties
		if prop, exists := macro.Properties[key]; exists {
			var sb strings.Builder
			sbSize := len(prop) + getValuesLen(values...)
			sb.Grow(sbSize)
			sb.WriteString(prop)
			// separate props with \n if there is more than one value
			if len(values) > 1 {
				sb.WriteString("\n")
			}
			sb.WriteString(values[0])
			macro.Properties[key] = sb.String()
		} else {
			macro.Properties[key] = values[0]
		}
	}
}

func getValuesLen(values ...string) int {
	l := 0
	for _, v := range values {
		l += len(v)
	}
	return l
}

func addMacroPropsToSect(ctx *config.Context, conf *Config, macroName *string) error {

	// Get current section
	cs := ctx.RunTime.Params[config.CurrSect]
	macro := ctx.RunTime.Macros[*macroName]

	// get config sectProps
	sectProps, sectExists := conf.Sects[cs]
	if !sectExists {
		sectProps = make(map[string]string)
	}
	for k, v := range macro.Properties {
		if v, err := applyParamsAndConstants(conf, v, &macro.Parameters); err != nil {
			return err
		} else {
			sectProps[k] = v
		}
	}
	conf.Sects[cs] = sectProps
	return nil
}

func addMacroPropsToMacro(ctx *config.Context, conf *Config, macroName *string) error {

	// Get current macro
	cm := ctx.RunTime.Params[config.CurrMacro]
	currMacro := ctx.RunTime.Macros[cm]
	useMacro := ctx.RunTime.Macros[*macroName]

	// add useMacro props to currMacro
	for k, v := range useMacro.Properties {
		if v, err := applyParamsAndConstants(conf, v, &useMacro.Parameters); err != nil {
			return err
		} else {
			currMacro.Properties[k] = v
		}
	}
	return nil
}

func applyParamsAndConstants(conf *Config, v string, params *map[string]string) (string, error) {
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
	if err := replaceConstants(sm, '-', 'p', conf); err != nil {
		return "", err
	}
	return sm.GetString('-', 'p'), nil
}
