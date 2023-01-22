package config

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"

	conf "github.com/grufgran/config/context"
)

type fileUnmarshaller struct {
	scanner           *bufio.Scanner
	data              *fileRowData
	inSkipSectionMode bool
}

func newFileUnmarshaller(f *os.File, absFilename string) *fileUnmarshaller {

	fum := fileUnmarshaller{
		scanner: bufio.NewScanner(f),
		data: &fileRowData{
			fileName: absFilename,
			fileDir:  filepath.Dir(absFilename),
		},
	}
	return &fum
}

// get next row to handle and update rowInfo. This is called from unMarshaller. Interface function
func (fum *fileUnmarshaller) scan() bool {
	status := fum.scanner.Scan()
	fum.data.row = fum.scanner.Text()
	fum.data.rowNumber++
	fum.data.findings = make(map[findingsType]string, 2)
	fum.data.prevRowType = fum.data.rowType
	fum.data.rowType = unknown
	return status
}

// preprocess data. This is called from unMarshaller. Interface function
func (fum *fileUnmarshaller) prepareData(ctx *conf.Context, conf *Config) error {

	// Set value and determine rowType
	err := fum.data.setValueAndType(ctx, conf, fum.inSkipSectionMode)
	// ignore (some) error if inSkipSectionMode
	if fum.inSkipSectionMode {
		switch fum.data.rowType {
		case unknown, comment, empty, multiLineHereDoc, multiLineBackslash, macroUse, property:
			fum.data.rowType = skipSection
			return nil
		}
	}
	if err != nil {
		return err
	}
	return nil
}

func (fum *fileUnmarshaller) setSkipSectionMode(newMode skipSectionMode) {
	if newMode == startSkipping {
		fum.inSkipSectionMode = true
	} else {
		fum.inSkipSectionMode = false
	}
}

func (fum *fileUnmarshaller) getFileRowData() *fileRowData {
	return fum.data
}

// create appropriate rowHandler. This is called from unMarshaller. Interface function
func (fum *fileUnmarshaller) getDataStrategy(ctx *conf.Context) dataStrategy {

	switch fum.data.rowType {
	// If we have found a section, we will investigate if it is a special section
	case section:
		return newSectStrategy(fum.data.value)

		// If we have found a skipSektion type, then its time to start skipping
	case skipSection:
		return newSkipSectionStrategy()

		// if we found a property or multiLineHereDoc, handle it properly
	case property, multiLineHereDoc, multiLineBackslash:
		key := fum.data.findings[key]
		value := fum.data.findings[value]
		ps := newPropertyStrategy(fum.data.rowType, key, value)
		return ps

		// if we found a include, then start read the new file
	case include, includeIfExist, includeIfExistWithBasePath:
		fileName := fum.data.findings[filePath]
		return newIncludeStrategy(fileName)

		// if we found a macroDefine, handle it properly
	case macroDefine:
		name := fum.data.findings[macroName]
		params := fum.data.findings[macroParams]
		return newMacroDefineStrategy(name, params)
		// if we found a macroDefine, handle it properly
	case macroUse:
		name := fum.data.findings[macroName]
		params := fum.data.findings[macroParams]
		numParams, _ := strconv.Atoi(fum.data.findings[numMacroParams])
		return newMacroUseStrategy(name, params, numParams)

	default:
		return &doNothingStrategy{}
	}
}

func readConfigFile(ctx *conf.Context, fileName string, conf *Config, logger *Logger) error {

	// get absolutepath for file
	absFilename, err := filepath.Abs(fileName)
	if err != nil {
		return err
	}

	// check so this file isn't already in the stack, avoiding an infinite loop
	if ctx.Stack.Contains(&absFilename) {
		return nil
	}

	// open file
	f, err := os.Open(fileName)
	if err != nil {
		return err
	}

	// remember to close the file at the end of the program
	defer f.Close()

	// set scanner
	fum := newFileUnmarshaller(f, absFilename)

	// Add filename to stack
	ctx.Stack.Push(absFilename)

	// Let the unmarshaller process the rows
	if err := unMarshall(ctx, fum, conf, logger); err != nil {
		return err
	}

	// Check status of scanner
	if err := fum.scanner.Err(); err != nil {
		return err
	}

	// pop info from ctx.stack
	ctx.Stack.Pop()
	return err
}
