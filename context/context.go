package context

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Context struct {
	BasePaths map[string]string
	Claims    claimSet
	Stack     stack
	RunTime   runTimeValues
}

// New ConfContext.
func NewContext(basePaths map[string]string, claims ...string) *Context {
	if basePaths == nil {
		basePaths = make(map[string]string)
	}
	ctx := Context{
		BasePaths: basePaths,
		Claims:    make(map[string]struct{}),
		Stack:     stack{},
		RunTime: runTimeValues{
			Params: make(map[paramType]string),
		},
	}
	ctx.AddClaims(claims...)
	return &ctx
}

// Add claims
func (ctx *Context) AddClaims(claims ...string) {
	for _, v := range claims {
		ctx.Claims[strings.TrimSpace(v)] = struct{}{}
	}
}

// Add paths. Like this pathName=path
func (ctx *Context) AddBasePaths(paths ...string) error {
	var sb strings.Builder
	for _, p := range paths {
		// split for example site=/site
		items := strings.SplitN(p, "=", 2)
		key := strings.TrimSpace(items[0])
		path := strings.TrimSpace(items[1])
		endsWithPathSeparator := strings.HasSuffix(path, string(os.PathSeparator))
		// convert path separators to slash. Nothing is changed on linux or mac. But windows paths are changed
		path = filepath.ToSlash(items[1])
		// make sure paths always ends with a slash
		if !endsWithPathSeparator {
			path = path + "/"
		}
		absPath, err := filepath.Abs(path)
		ctx.BasePaths[key] = absPath
		if err != nil {
			if sb.Len() > 0 {
				sb.WriteRune('\n')
			}
			sb.WriteString(path)
			sb.WriteString(", could not translate ")
			sb.WriteString(path)
			sb.WriteString(" to absolute path")
		}
	}
	if sb.Len() > 0 {
		return fmt.Errorf("%v", sb.String())
	}
	return nil
}

func (ctx *Context) GetBasePath(key string) (string, error) {
	if basePath, exists := ctx.BasePaths[key]; exists {
		return basePath, nil
	} else {
		err := fmt.Errorf("base path key %v, not defined in confContext", key)
		return "", err
	}
}

func isConfRoot(s *string) bool {
	counter := 0
	for _, r := range *s {
		counter++
		if counter == 1 && r != '^' {
			return false
		} else if counter == 2 && r != '/' {
			return false
		} else {
			return true
		}
	}
	return false
}

func (ctx *Context) GetConfRoot() (string, error) {
	if confRoot, exists := ctx.BasePaths["^/"]; exists {
		return confRoot, nil
	} else {
		err := fmt.Errorf("config root not defined in confContext")
		return "", err
	}
}

func (ctx *Context) SetConfRoot(confRoot string) error {

	// check if confRoot is a file, then strip the filename
	if i, err := os.Stat(confRoot); err == nil {
		if !i.IsDir() {
			confRoot = filepath.Dir(confRoot)
		}
	} else {
		return fmt.Errorf("confRoot=%v, not found", confRoot)
	}

	// convert path separators to slash. Nothing is changed on linux or mac. But windows paths are changed
	path := filepath.ToSlash(confRoot)
	if absPath, err := filepath.Abs(path); err == nil {
		// make sure paths always ends with a slash
		ctx.BasePaths["^/"] = absPath + "/"
	} else {
		return fmt.Errorf("confRoot=%v, could not translate %v to absolute path", confRoot, path)
	}
	return nil
}

// Check if fileName exists
func (ctx *Context) CheckIfFileExists(fileDir, fileName, basePath *string) (bool, *string, error) {

	var sb strings.Builder
	// filename starting with ^/ means, use ctx.ConfRoot
	if isConfRoot(fileName) {
		confRoot, err := ctx.GetConfRoot()
		if err != nil {
			return false, fileName, err
		}
		sb.Grow(len(confRoot) + len(*fileName))
		sb.WriteString(confRoot)
		sb.WriteString((*fileName)[2:])
	} else if basePath != nil {
		// if basePath is provided, then use that instead of fileDir
		sb.Grow(len(*basePath) + len(*fileName))
		sb.WriteString(*basePath)
		sb.WriteString(*fileName)
	} else {
		// use fileDir
		sb.Grow(len(*fileDir) + len(*fileName) + 1)
		sb.WriteString(*fileDir)
		sb.WriteRune(os.PathSeparator)
		sb.WriteString(*fileName)
	}
	if filePath, err := filepath.Abs(sb.String()); err != nil {
		return false, fileName, err
	} else {
		if _, err := os.Stat(filePath); errors.Is(err, os.ErrNotExist) {
			return false, &filePath, nil
		} else {
			// file exists, return abs path
			return true, &filePath, nil
		}
	}
}

// get the folder of the executable
func (ctx *Context) GetExeFolder() string {
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	exPath := filepath.Dir(ex)
	return exPath
}
