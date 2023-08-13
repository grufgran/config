package config

import (
	"fmt"
	"strconv"
	"strings"

	config "github.com/grufgran/config/context"
	"github.com/grufgran/config/stringMask"
)

type dataType int8
type findingsType int8

const (
	unknown dataType = iota
	comment
	empty
	multiLineHereDoc
	multiLineBackslash
	section
	skipSection
	include
	includeIfExist
	includeIfExistWithBasePath
	macroUse
	macroDefine
	property

	key findingsType = iota
	value
	filePath
	macroName
	macroParams
	numMacroParams
)

type fileRowData struct {
	fileName    string
	fileDir     string
	row         string
	rowNumber   int
	value       string
	rowType     dataType
	prevRowType dataType
	findings    map[findingsType]string
}

// mask row and set rowType
func (frd *fileRowData) setValueAndType(ctx *config.Context, conf *Config, inSkipSectionMode bool) error {

	// Init cmask
	sm := stringMask.NewStringMask(frd.row, '-', '#')

	// if we have an ending string like this  "....\ # comment", we need to mask white space after the backslash.
	// we only do this, when there is a comment marker. An ending backslash means multiline. But then the backslash
	// has to be in the *last* position of the string, otherwise it is not a multiline.
	// So by adding a space after the backslash, can be a sign from the user, that multiline is not wanted.
	// The user is trying to tell us that this is just a normal property ending with a backslash. Not a multiline.
	// Of course it can be a result of bad editing also, but that we can not know.
	// first determine if the row has a comment marker
	if sm.Comment.Pos > 0 {
		// mask white trailing space
		sm.MaskEndSpaces('X', '-')

	} else if sm.Comment.Pos == 0 {
		// we have a comment string, like this: "# this is a comment"
		frd.rowType = comment
		frd.value = ""
		return nil
	}

	// Get endPoint for later use
	lastPoint := sm.GetLastMaskPoint('-')

	// if we are inMultiLineMode, special rules apply. Then we just replace constants
	// If the previous row had a backslash, then we are in multiline mode. Even if this paricular row doesn't have a backslash.
	if frd.prevRowType == multiLineBackslash || frd.prevRowType == multiLineHereDoc {

		// in multiLineHereDoc mode, #-characters are allowed. I e they are not comment marks, so recreate the stringmask
		if frd.prevRowType == multiLineHereDoc && sm.Comment.Pos != -1 {
			sm = stringMask.NewStringMask(frd.row, '-')
		}

		// mask and replace constants.
		// But, if we are in macroDefine mode, no constants shall be replaces. Those constants will be replaces during macroUse
		if ctx.RunTime.SaveTo != config.Macros {
			if err := conf.replaceConstants(sm, '-', 'C'); err != nil {
				return err
			}
		}

		// Mask ending backslash if such backslash exists
		if frd.prevRowType == multiLineBackslash && lastPoint != nil {
			// is the last rune a backslash?
			if lastPoint.Rune == '\\' {
				sm.MaskAtPos(lastPoint.Pos, '\\')
				frd.rowType = multiLineBackslash
			}
		}
		// set value and exit
		frd.value = sm.GetString('-', 'C')
		return nil
	}

	// This is a string with some white spaces and a comment. For example: "  # my comment"
	if lastPoint == nil {
		frd.rowType = empty
		frd.value = ""
		return nil
	}

	// mask trailing and leading white space
	sm.MaskStartEndSpaces('X', '-')

	// get first and last rune where mask is '-'
	startPoint := sm.GetFirstMaskPoint('-')
	endPoint := sm.GetLastMaskPoint('-')
	// If the runes are [ and ] then this is a section
	if startPoint.Rune == '[' && endPoint.Rune == ']' {
		// mask first [ and last ]
		sm.MaskAtPos(startPoint.Pos, '[')
		sm.MaskAtPos(endPoint.Pos, ']')

		// mask and get claims
		if claimsFulfilled, err := conf.hasRequiredClaims(sm, '-', 'c', ctx); err != nil {
			// return the error
			return err
		} else if !claimsFulfilled {
			// if not required claims are present, we have to skip this section
			frd.value = frd.row
			frd.rowType = skipSection
			return nil
		}

		// Trim white space around []-runes
		sm.MaskRightSpacesFromPos(startPoint.Pos+1, 'X', '-')
		sm.MaskLeftSpacesFromPos(endPoint.Pos-1, 'X', '-')
		frd.rowType = section
		frd.value = sm.GetString('-')

		// check if there is a =-sign and frd.value starts with include
		if equalSign := sm.MaskFirst('=', '=', '-'); equalSign != nil {
			if strings.HasPrefix(frd.value, "include") {
				sm.MaskLeftRightSpacesAround(equalSign.Pos, 'X', '-')
				if err := frd.handleIncludes(ctx, sm); err != nil {
					return err
				}
			}
			// check if this is a macro define
		} else if strings.HasPrefix(frd.value, "define ") {
			if err := frd.extractMacroNameAndParams("define ", sm); err != nil {
				return err
			}
			frd.rowType = macroDefine
			// check if this is a macro use
		} else if strings.HasPrefix(frd.value, "use ") {
			if err := frd.extractMacroNameAndParams("use ", sm); err != nil {
				return err
			}
			frd.rowType = macroUse
		}
		return nil
	}

	// if we are in inSkipSectionMode, there is no need to go further, since this is obvious not a section.
	// Sections are the only type that can break the inSkipSectionMode
	if inSkipSectionMode {
		frd.rowType = unknown
		return nil
	}

	// If there is a equal sign, then this must be a property
	equalSign := sm.MaskFirst('=', '=', '-')
	if equalSign == nil {
		frd.rowType = unknown
		frd.value = sm.GetString('-')
		return fmt.Errorf("unkonwn rowtype: %v, at rownumber %v in file %v", frd.row, frd.rowNumber, frd.fileName)
	} else {
		frd.rowType = property
	}
	// Trim white space around =-char
	sm.MaskLeftRightSpacesAround(equalSign.Pos, 'X', '-')

	// check if the last rune is a backslash, since we are not currently in mulitline mode, this must be the first row of the multiline
	if lastPoint.Rune == '\\' {
		sm.MaskAtPos(lastPoint.Pos, '\\')
		frd.rowType = multiLineBackslash
	}

	// mask and replace constants
	// But, if we are in macroDefine mode, no constants shall be replaces. Those constants will be replaces during macroUse
	if ctx.RunTime.SaveTo != config.Macros {
		if err := conf.replaceConstants(sm, '-', 'C'); err != nil {
			return err
		}
	}
	kvp := sm.GetStrings('-', 'C')
	frd.findings[key] = kvp[0]
	// check if kvp[1] is a hereDocMarker
	if isHereDocType(&kvp[1]) {
		frd.rowType = multiLineHereDoc
		frd.findings[value] = kvp[1][2:]
	} else {
		frd.findings[value] = kvp[1]
	}
	sb := strings.Builder{}
	sb.WriteString(kvp[0])
	sb.WriteRune('=')
	sb.WriteString(kvp[1])
	frd.value = sb.String()

	return nil
}

func isHereDocType(s *string) bool {
	counter := 0
	for _, r := range *s {
		counter++
		if r == '<' {
			if counter == 2 {
				return true
			}
		} else {
			return false
		}
	}
	return false
}

func (frd *fileRowData) handleIncludes(ctx *config.Context, sm *stringMask.StringMask) error {

	// get the include type and the file to read
	items := sm.GetStrings('-')
	switch items[0] {
	case "include":
		frd.rowType = include
	case "include_if_exists":
		frd.rowType = includeIfExist
	default:
		frd.rowType = includeIfExistWithBasePath
	}

	// check if basePath is provided when we have a includeIfExistWithBasePath
	basePath, err := frd.getBasePath(ctx, &items[0])
	if err != nil {
		return err
	}
	// check if the file exists
	if fileExists, fileName, err := ctx.CheckIfFileExists(&frd.fileDir, &items[1], basePath); err != nil {
		return err
	} else if fileExists {
		frd.findings[filePath] = *fileName
	} else {
		// if it is an include=someFile, we must return an error if the file doesn't exsist
		if frd.rowType == include {
			return fmt.Errorf("file %s not found from %s", *fileName, frd.value)
		}
		// when it is include_if_exists or include_site_if_exists, then we just skips the section
		frd.rowType = skipSection
	}
	return nil
}

func (frd *fileRowData) getBasePath(ctx *config.Context, key *string) (*string, error) {

	if frd.rowType == includeIfExistWithBasePath {
		// extract basepath from string include_<basepath>_if_exists=
		var sb strings.Builder
		for _, r := range (*key)[8:] {
			if r == '_' {
				break
			}
			sb.WriteRune(r)
		}
		if basePath, err := ctx.GetBasePath(sb.String()); err != nil {
			return nil, err
		} else {
			return &basePath, nil
		}

	}
	return nil, nil
}

func (frd *fileRowData) extractMacroNameAndParams(identifier string, sm *stringMask.StringMask) error {

	// example [define myMacro($y, $z)]
	// example [use myMacro(1, 2)]
	// or [use myMacro(1, "2")]
	ident := sm.MaskFirstWord(identifier, '#', '-')
	sm.MaskRightSpacesFromPos(ident.Pos+len(identifier), 'X', '-')

	// mask all "-chars. Because a parameter could look like this:
	// [use myMacro(1, "2,)")]
	quots := sm.Mask('"', '"', '-')
	// there must be even num of quots
	if len(*quots) > 0 {
		if len(*quots)%2 != 0 {
			return fmt.Errorf("macro definition with uneven number of quotation marks: %v", frd.row)
		}
		// mask everything between them
		for i := 0; i < len(*quots); i += 2 {
			sm.MaskBetween((*quots)[i].Pos+1, (*quots)[i+1].Pos-1, 'q')
		}
	}

	// Mask parenthesis
	leftPar := sm.MaskFirst('(', '(', '-')
	rightPar := sm.MaskLast(')', ')', '-')

	// check if we got two parenthesis
	if leftPar != nil && rightPar != nil {
		sm.MaskRightSpacesFromPos(leftPar.Pos+1, 'X', '-')
		sm.MaskLeftSpacesFromPos(rightPar.Pos-1, 'X', '-')

		// find all commas, and trim space around them
		commas := sm.Mask(',', 'd', '-')
		sm.MaskLeftRightSpacesAroundPoints(commas, 'X', '-')

		// unmask between quotation marks, if there were some
		if len(*quots) > 0 {
			// mask everything between them
			for i := 0; i < len(*quots); i += 2 {
				sm.MaskBetween((*quots)[i].Pos+1, (*quots)[i+1].Pos-1, '-')
			}
		}
		// get macroname and params
		items := sm.GetStrings('-')
		frd.findings[macroName] = items[0]

		// create params
		frd.findings[numMacroParams] = strconv.Itoa(len(items) - 1)
		if len(items) > 1 {
			frd.findings[macroParams] = strings.Join(items[1:], string(rune(0)))
		}
	} else if leftPar == nil && rightPar == nil {
		// macro without parenthesis
		frd.findings[macroName] = sm.GetString('-')
	} else if leftPar == nil {
		return fmt.Errorf("macro definition without left parenthesis: %v", frd.row)
	} else {
		return fmt.Errorf("macro definition without right parenthesis: %v", frd.row)
	}
	return nil
}
