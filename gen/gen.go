package gen

import (
	"bytes"
	"embed"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/hysios/mx/logger"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

type FileSystem struct {
	Root             string
	Verbose          bool
	Contents         embed.FS
	Files            map[string]*template.Template
	aftersCall       []func(ctx Context) error
	beforesCall      []func(ctx Context) error
	Ignores          []string
	Funcs            template.FuncMap
	Variables        map[string]interface{}
	locals           map[string]map[string]interface{}
	inhibits         []string
	fileIndexs       []string
	fileTypeContexts map[string]ctxCtor
	fileContexts     map[string]ctxCtor
	logger           *zap.Logger
	// contexts map[string]
}

type Output struct {
	Directory string
	Verbose   bool
	Overwrite bool
}

func (fs *FileSystem) init() {
	if fs.logger == nil {
		fs.logger = logger.Cli
	}
}

func (fs *FileSystem) Gen(output *Output) error {
	fs.init()

	oldvebose := fs.Verbose

	defer func() {
		fs.Verbose = oldvebose
	}()
	fs.Verbose = output.Verbose
	if output.Verbose {
		fs.logger, _ = zap.NewDevelopment(zap.IncreaseLevel(zap.DebugLevel))
	}
	fs.AddVariable("OutputDir", output.Directory)
	// call before
	for _, call := range fs.beforesCall {
		if err := call(&BaseContext{
			vars: fs.Variables,
		}); err != nil {
			return err
		}
	}

	// generate files
	if err := fs.Range(func(name string, file *IterFile) error {
		fs.logger.Debug("generate file", zap.String("name", name))
		switch file.Mode {
		case ModeFile:
			return fs.CopyTo(output.Directory, name, file.Rd, output.Overwrite)
		case ModeTemplate:
			ctx, err := fs.buildContext(name, file)
			if err != nil {
				return err
			}

			ctx.Scan()

			return fs.ExecuteTo(output.Directory, name, file.Tmpl, output.Overwrite, ctx)
		}
		return nil
	}); err != nil {
		return err
	}

	// call after
	for _, call := range fs.aftersCall {
		if err := call(&BaseContext{
			vars: fs.Variables,
		}); err != nil {
			return err
		}
	}

	return nil
}

func (fs *FileSystem) SetLogger(logger *zap.Logger) {
	fs.logger = logger
}

func (fs *FileSystem) Before(call func(ctx Context) error) {
	fs.beforesCall = append(fs.beforesCall, call)
}

func (fs *FileSystem) After(call func(ctx Context) error) {
	fs.aftersCall = append(fs.aftersCall, call)
}

// AddFile adds a file to the file system.
func (fs *FileSystem) AddFile(name string, tmpl *template.Template) {
	if fs.Files == nil {
		fs.Files = make(map[string]*template.Template)
	}
	fs.Files[name] = tmpl
	fs.fileIndexs = append(fs.fileIndexs, name)
}

func (fs *FileSystem) MustParse(name string) *template.Template {
	tmpl, err := template.ParseFS(fs.Contents, name)
	if err != nil {
		panic(err)
	}

	fs.inhibits = append(fs.inhibits, name)
	return tmpl
}

// AddFuncMap adds a function map to the file system.
func (fs *FileSystem) AddFuncMap(name string, funcMap template.FuncMap) {
	if fs.Funcs == nil {
		fs.Funcs = make(template.FuncMap)
	}
	fs.Funcs[name] = funcMap
}

// AddVariable adds a global variable to the file system.
func (fs *FileSystem) AddVariable(name string, value interface{}) {
	if fs.Variables == nil {
		fs.Variables = make(map[string]interface{})
	}
	fs.Variables[name] = value
}

func (fs *FileSystem) AddLocal(filename string, variable string, value interface{}) {
	if fs.locals == nil {
		fs.locals = make(map[string]map[string]interface{})
	}
	if fs.locals[filename] == nil {
		fs.locals[filename] = make(map[string]interface{})
	}

	fs.locals[filename][variable] = value
}

func (fs *FileSystem) AddIgnore(name string) {
	fs.Ignores = append(fs.Ignores, name)
}

// Range calls f sequentially for each file in the file system.
// If f returns an error, range stops the iteration and returns the error.
func (fs *FileSystem) Range(f func(name string, file *IterFile) error) error {
	var errs error
	if err := fs.iterate(fs.Root, f); err != nil {
		errs = multierr.Append(errs, err)
	}

	for _, filepattrn := range fs.fileIndexs {
		file := fs.buildFilename(filepattrn)
		if err := f(file, &IterFile{
			Mode:        ModeTemplate,
			Tmpl:        fs.Files[filepattrn],
			FilePattern: filepattrn,
		}); err != nil {
			errs = multierr.Append(errs, err)
			continue
		}
	}

	return errs
}

const (
	ModeTemplate = "template"
	ModeFile     = "file"
)

type IterFile struct {
	Mode        string
	FilePattern string
	Tmpl        *template.Template
	Rd          io.Reader
}

// iterate iterates over the file system.
func (fs *FileSystem) iterate(dir string, f func(name string, file *IterFile) error) error {
	files, err := fs.Contents.ReadDir(dir)
	if err != nil {
		return err
	}
	var errs error

Next:
	for _, file := range files {
		if file.IsDir() {
			fs.iterate(filepath.Join(dir, file.Name()), f)
			continue
		}

		for _, ignore := range fs.Ignores {
			if matched, _ := filepath.Match(ignore, filepath.Join(dir, file.Name())); matched {
				continue Next
			}
		}

		for _, inhibit := range fs.inhibits {
			if matched, _ := filepath.Match(inhibit, filepath.Join(dir, file.Name())); matched {
				continue Next
			}
		}

		// file ext is .tmpl
		if filepath.Ext(file.Name()) == ".tmpl" {
			tmpl, err := template.ParseFS(fs.Contents, filepath.Join(dir, file.Name()))
			if err != nil {
				return err
			}

			// extract .tmpl ext filename
			name := strings.TrimSuffix(file.Name(), ".tmpl")
			filename := fs.extractRootPath(dir, name)
			if err := f(filename, &IterFile{
				Mode: ModeTemplate,
				Tmpl: tmpl,
				Rd:   nil,
			}); err != nil {
				errs = multierr.Append(errs, err)
				continue
			}
		} else {
			_f, err := fs.Contents.Open(filepath.Join(dir, file.Name()))
			if err != nil {
				return err
			}

			// extract root prefix path
			filename := fs.extractRootPath(dir, file.Name())
			if err := f(filename, &IterFile{
				Mode: ModeFile,
				Rd:   _f,
			}); err != nil {
				errs = multierr.Append(errs, err)
				continue
			}
		}

	}

	return nil
}

// extractRootPath
func (fs *FileSystem) extractRootPath(dir string, filename string) string {
	return strings.TrimPrefix(strings.TrimPrefix(filepath.Join(dir, filename), fs.Root), "/")
}

// CopyTo copies a file to the specified directory.
func (fs *FileSystem) CopyTo(directory string, name string, rd io.Reader, overwrite bool) error {
	// create directory

	if err := os.MkdirAll(filepath.Dir(filepath.Join(directory, name)), 0755); err != nil {
		return err
	}

	// check file exists
	if _, err := os.Stat(filepath.Join(directory, name)); err == nil {
		if !overwrite {
			return ErrFileExists
		}
	}

	// create file
	f, err := os.OpenFile(filepath.Join(directory, name), os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	// copy file
	if _, err := io.Copy(f, rd); err != nil {
		return err
	}

	return nil
}

// ExecuteTo executes a template to the specified directory.
func (fs *FileSystem) ExecuteTo(directory string, name string, tmpl *template.Template, overwrite bool, data interface{}) (err error) {
	// create directory
	// recover error
	defer func() {
		if _err, ok := recover().(error); ok {
			err = _err
		}
	}()

	if err := os.MkdirAll(filepath.Dir(filepath.Join(directory, name)), 0755); err != nil {
		return err
	}

	// check file exists
	if _, err := os.Stat(filepath.Join(directory, name)); err == nil {
		if !overwrite {
			return ErrFileExists
		}
	}

	var w io.Writer
	// create file
	f, err := os.OpenFile(filepath.Join(directory, name), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	if fs.Verbose {
		w = io.MultiWriter(f, os.Stdout)
	} else {
		w = f
	}
	// execute template
	if err := tmpl.Execute(w, data); err != nil {
		return err
	}

	return nil
}

// buildContext
func (fs *FileSystem) buildContext(name string, file *IterFile) (ctx Context, err error) {
	var ext = filepath.Ext(name)

	// copy fs variable to context
	variable := make(map[string]interface{})
	for k, v := range fs.Variables {
		variable[k] = v
	}

	for k, v := range fs.locals[name] {
		variable[k] = v
	}

	ctx = &BaseContext{
		vars: variable,
	}

	if extctor, ok := fs.fileTypeContexts[ext]; ok {
		ctx, err = extctor(ctx)
		if err != nil {
			return nil, err
		}
	}

	if file.FilePattern != "" {
		if ctor, ok := fs.fileContexts[file.FilePattern]; ok {
			ctx, err = ctor(ctx)
			if err != nil {
				return nil, err
			}
		}
	}

	if ctor, ok := fs.fileContexts[name]; ok {
		ctx, err = ctor(ctx)
		if err != nil {
			return nil, err
		}
	}

	return
}

// buildFilename
func (fs *FileSystem) buildFilename(name string) string {
	tmpl := template.Must(template.New("filename").Parse(name))
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, fs.Variables); err != nil {
		return name
	}

	return buf.String()
}

func (fs *FileSystem) AddFileTypeContext(extfile string, f func(baseCtx Context) (Context, error)) {
	if fs.fileTypeContexts == nil {
		fs.fileTypeContexts = make(map[string]ctxCtor)
	}

	fs.fileTypeContexts[extfile] = f
}

func (fs *FileSystem) AddFileContext(name string, f func(baseCtx Context) (Context, error)) {
	if fs.fileContexts == nil {
		fs.fileContexts = make(map[string]ctxCtor)
	}
	fs.fileContexts[name] = f
}
