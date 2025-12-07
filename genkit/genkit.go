// Package genkit provides a framework for building Go code generators,
// inspired by google.golang.org/protobuf/compiler/protogen and Go's standard toolchain.
//
// Key design principles:
//   - Library-first: designed as a library, not a plugin system
//   - Go toolchain compatible: supports "./..." patterns like go build
//   - Type-safe API: structured types for describing source code elements
//   - GeneratedFile abstraction: convenient code generation with automatic import management
//
// Basic usage:
//
//	gen := genkit.New()
//	if err := gen.Load("./..."); err != nil {
//	    log.Fatal(err)
//	}
//	for _, pkg := range gen.Packages {
//	    for _, enum := range pkg.Enums {
//	        g := gen.NewGeneratedFile(genkit.OutputPath(pkg.Dir, enum.Name+"_enum.go"))
//	        // generate code...
//	    }
//	}
//	if err := gen.Write(); err != nil {
//	    log.Fatal(err)
//	}
package genkit

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"go/types"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"
)

// Generator is the main entry point for code generation.
type Generator struct {
	// Packages are the loaded packages.
	Packages []*Package

	// Fset is the token file set.
	Fset *token.FileSet

	generatedFiles []*GeneratedFile
	opts           Options
}

// Options configures the generator.
type Options struct {
	// Tags are build tags to use when loading packages.
	Tags []string

	// Dir is the working directory. If empty, uses current directory.
	Dir string

	// IgnoreGeneratedFiles when true, ignores files that start with
	// "// Code generated" comment. This is useful for ignoring generated
	// files that may have syntax errors.
	IgnoreGeneratedFiles bool
}

// New creates a new Generator.
func New(opts ...Options) *Generator {
	g := &Generator{
		Fset: token.NewFileSet(),
	}
	if len(opts) > 0 {
		g.opts = opts[0]
	}
	return g
}

// Load loads packages matching the given patterns.
// Patterns follow Go's standard conventions:
//   - "./..."  - current directory and all subdirectories
//   - "./pkg"  - specific package
//   - "."      - current directory only
func (g *Generator) Load(patterns ...string) error {
	cfg := &packages.Config{
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedCompiledGoFiles |
			packages.NeedImports |
			packages.NeedTypes |
			packages.NeedSyntax |
			packages.NeedTypesInfo,
		Fset:       g.Fset,
		Dir:        g.opts.Dir,
		BuildFlags: buildFlags(g.opts.Tags),
	}

	pkgs, err := packages.Load(cfg, patterns...)
	if err != nil {
		return fmt.Errorf("load packages: %w", err)
	}

	// Check for package errors, but ignore errors from files matching IgnoreFiles
	var errs []error
	for _, pkg := range pkgs {
		for _, e := range pkg.Errors {
			if g.shouldIgnoreError(e) {
				continue
			}
			errs = append(errs, e)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("package errors: %v", errs)
	}

	// Build our Package types
	for _, pkg := range pkgs {
		g.Packages = append(g.Packages, g.buildPackage(pkg))
	}

	return nil
}

// shouldIgnoreError checks if the error should be ignored based on IgnoreGeneratedFiles.
func (g *Generator) shouldIgnoreError(e packages.Error) bool {
	if !g.opts.IgnoreGeneratedFiles {
		return false
	}

	// Try to extract filename from Pos first (format: "/path/to/file.go:line:col")
	if filename := extractFilename(e.Pos); filename != "" && isGeneratedFile(g.resolveFilename(filename)) {
		return true
	}

	// Also check Msg which may contain relative path (format: "path/to/file.go:line:col: error")
	if filename := extractFilenameFromMsg(e.Msg); filename != "" && isGeneratedFile(g.resolveFilename(filename)) {
		return true
	}

	return false
}

// resolveFilename resolves a potentially relative filename to absolute path.
func (g *Generator) resolveFilename(filename string) string {
	if filepath.IsAbs(filename) {
		return filename
	}
	if g.opts.Dir != "" {
		return filepath.Join(g.opts.Dir, filename)
	}
	// Try current working directory
	if abs, err := filepath.Abs(filename); err == nil {
		return abs
	}
	return filename
}

// extractFilename extracts filename from position string (format: "file.go:line:col")
func extractFilename(pos string) string {
	if pos == "" {
		return ""
	}
	if idx := strings.Index(pos, ":"); idx > 0 {
		return pos[:idx]
	}
	return pos
}

// extractFilenameFromMsg extracts filename from error message.
// Message format may be: "# pkg\npath/to/file.go:line:col: error"
func extractFilenameFromMsg(msg string) string {
	lines := strings.Split(msg, "\n")
	for _, line := range lines {
		// Look for pattern "file.go:line:col:"
		if strings.Contains(line, ".go:") {
			if idx := strings.Index(line, ".go:"); idx >= 0 {
				return line[:idx+3] // include ".go"
			}
		}
	}
	return ""
}

// NewGeneratedFile creates a new file to be generated.
func (g *Generator) NewGeneratedFile(filename string, importPath GoImportPath) *GeneratedFile {
	gf := &GeneratedFile{
		filename:      filename,
		goImportPath:  importPath,
		buf:           new(bytes.Buffer),
		imports:       make(map[GoImportPath]*importInfo),
		usedPackages:  make(map[GoPackageName]GoImportPath),
		manualImports: make(map[GoImportPath]GoPackageName),
	}
	g.generatedFiles = append(g.generatedFiles, gf)
	return gf
}

// Write writes all generated files to disk.
func (g *Generator) Write() error {
	for _, gf := range g.generatedFiles {
		if gf.skip {
			continue
		}
		content, err := gf.Content()
		if err != nil {
			return fmt.Errorf("generate %s: %w", gf.filename, err)
		}
		if content == nil {
			continue
		}

		dir := filepath.Dir(gf.filename)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create dir %s: %w", dir, err)
		}
		if err := os.WriteFile(gf.filename, content, 0644); err != nil {
			return fmt.Errorf("write %s: %w", gf.filename, err)
		}
	}
	return nil
}

// DryRun returns generated content without writing files.
func (g *Generator) DryRun() (map[string][]byte, error) {
	result := make(map[string][]byte)
	for _, gf := range g.generatedFiles {
		if gf.skip {
			continue
		}
		content, err := gf.Content()
		if err != nil {
			return nil, err
		}
		if content != nil {
			result[gf.filename] = content
		}
	}
	return result, nil
}

func (g *Generator) buildPackage(pkg *packages.Package) *Package {
	// Filter out ignored files from GoFiles
	var goFiles []string
	for _, f := range pkg.GoFiles {
		if !g.shouldIgnoreFile(f) {
			goFiles = append(goFiles, f)
		}
	}

	// Filter out ignored files from Syntax
	var syntax []*ast.File
	for _, file := range pkg.Syntax {
		if file == nil {
			continue
		}
		pos := g.Fset.Position(file.Pos())
		if !g.shouldIgnoreFile(pos.Filename) {
			syntax = append(syntax, file)
		}
	}

	p := &Package{
		Name:      pkg.Name,
		PkgPath:   pkg.PkgPath,
		Dir:       pkgDir(pkg),
		GoFiles:   goFiles,
		Fset:      g.Fset,
		TypesPkg:  pkg.Types,
		TypesInfo: pkg.TypesInfo,
		Syntax:    syntax,
	}

	// First pass: collect all type declarations
	typesByName := make(map[string]*Type)
	for _, file := range syntax {
		g.extractTypes(p, file, typesByName)
	}

	// Second pass: extract enum constants and link to types
	for _, file := range syntax {
		g.extractEnums(p, file, typesByName)
	}

	return p
}

// shouldIgnoreFile checks if the file should be ignored based on IgnoreGeneratedFiles.
func (g *Generator) shouldIgnoreFile(filename string) bool {
	if !g.opts.IgnoreGeneratedFiles {
		return false
	}
	return isGeneratedFile(filename)
}

// isGeneratedFile checks if a file starts with "// Code generated" comment.
func isGeneratedFile(filename string) bool {
	f, err := os.Open(filename)
	if err != nil {
		return false
	}
	defer f.Close() //nolint:errcheck

	// Read first 256 bytes - enough to check for generated file header
	buf := make([]byte, 256)
	n, err := f.Read(buf)
	if err != nil || n == 0 {
		return false
	}

	content := string(buf[:n])
	// Check if file starts with "// Code generated" (standard Go convention)
	return strings.HasPrefix(content, "// Code generated")
}

func (g *Generator) extractTypes(pkg *Package, file *ast.File, typesByName map[string]*Type) {
	for _, decl := range file.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.TYPE {
			continue
		}

		for _, spec := range gd.Specs {
			ts := spec.(*ast.TypeSpec)
			typ := &Type{
				Name:     ts.Name.Name,
				Doc:      docText(gd.Doc),
				Pkg:      pkg,
				TypeSpec: ts,
			}

			// Extract struct fields
			if st, ok := ts.Type.(*ast.StructType); ok {
				typ.Fields = extractFields(g.Fset, st)
			}

			pkg.Types = append(pkg.Types, typ)
			typesByName[typ.Name] = typ
		}
	}
}

func (g *Generator) extractEnums(pkg *Package, file *ast.File, typesByName map[string]*Type) {
	for _, decl := range file.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.CONST {
			continue
		}

		// Group constants by type
		enumsByType := make(map[string]*Enum)
		var currentType string

		for _, spec := range gd.Specs {
			vs := spec.(*ast.ValueSpec)

			// Update current type if specified
			if vs.Type != nil {
				currentType = exprString(vs.Type)
			}
			if currentType == "" {
				continue
			}

			enum := enumsByType[currentType]
			if enum == nil {
				// Get doc from the type declaration if available
				doc := ""
				if typ, ok := typesByName[currentType]; ok {
					doc = typ.Doc
				}
				enum = &Enum{
					Name: currentType,
					Doc:  doc,
					Pkg:  pkg,
				}
				enumsByType[currentType] = enum
			}

			for i, name := range vs.Names {
				ev := &EnumValue{
					Name:    name.Name,
					Doc:     docText(vs.Doc),
					Comment: commentText(vs.Comment),
					Pos:     g.Fset.Position(name.Pos()),
				}
				if i < len(vs.Values) {
					ev.Value = exprString(vs.Values[i])
				}
				enum.Values = append(enum.Values, ev)
			}
		}

		for _, enum := range enumsByType {
			if len(enum.Values) > 0 {
				pkg.Enums = append(pkg.Enums, enum)
			}
		}
	}
}

// GeneratedFile represents a file to be generated.
type GeneratedFile struct {
	filename      string
	goImportPath  GoImportPath
	buf           *bytes.Buffer
	imports       map[GoImportPath]*importInfo
	usedPackages  map[GoPackageName]GoImportPath
	manualImports map[GoImportPath]GoPackageName
	skip          bool
}

type importInfo struct {
	importPath  GoImportPath
	packageName GoPackageName
}

// GoPrintable is implemented by types that can print themselves to a GeneratedFile.
type GoPrintable interface {
	PrintTo(g *GeneratedFile)
}

// P prints a line to the generated file.
// Arguments are concatenated without spaces. Use GoIdent for automatic import handling.
// Special types: GoIdent, GoMethod, GoFunc, GoDoc, GoParams, GoResults are formatted appropriately.
func (g *GeneratedFile) P(v ...any) {
	for _, x := range v {
		g.print(x)
	}
	g.buf.WriteByte('\n')
}

func (g *GeneratedFile) print(v any) {
	switch v := v.(type) {
	case string:
		g.buf.WriteString(v)
	case GoIdent:
		g.buf.WriteString(g.QualifiedGoIdent(v))
	case *GoIdent:
		g.buf.WriteString(g.QualifiedGoIdent(*v))
	case GoPrintable:
		v.PrintTo(g)
	default:
		fmt.Fprint(g.buf, v)
	}
}

// QualifiedGoIdent returns the qualified identifier string with import handling.
func (g *GeneratedFile) QualifiedGoIdent(ident GoIdent) string {
	if ident.GoImportPath == g.goImportPath || ident.GoImportPath == "" {
		return ident.GoName
	}
	return string(g.goPackageName(ident.GoImportPath)) + "." + ident.GoName
}

// Import explicitly imports a package and returns its local name.
func (g *GeneratedFile) Import(importPath GoImportPath) GoPackageName {
	return g.goPackageName(importPath)
}

// ImportAs explicitly imports a package with a custom alias.
// This is useful for packages where the import path doesn't match the package name,
// e.g., "gopkg.in/yaml.v3" should be imported as "yaml".
func (g *GeneratedFile) ImportAs(importPath GoImportPath, alias GoPackageName) GoPackageName {
	g.manualImports[importPath] = alias
	return g.goPackageName(importPath)
}

func (g *GeneratedFile) goPackageName(importPath GoImportPath) GoPackageName {
	if info, ok := g.imports[importPath]; ok {
		return info.packageName
	}

	if name, ok := g.manualImports[importPath]; ok {
		info := &importInfo{importPath: importPath, packageName: name}
		g.imports[importPath] = info
		g.usedPackages[name] = importPath
		return name
	}

	baseName := GoPackageName(path.Base(string(importPath)))
	name := baseName

	// Handle conflicts
	for i := 2; ; i++ {
		if existing, ok := g.usedPackages[name]; !ok || existing == importPath {
			break
		}
		name = GoPackageName(fmt.Sprintf("%s%d", baseName, i))
	}

	info := &importInfo{importPath: importPath, packageName: name}
	g.imports[importPath] = info
	g.usedPackages[name] = importPath
	return name
}

// Skip marks this file to be skipped.
func (g *GeneratedFile) Skip() { g.skip = true }

// Unskip reverses Skip.
func (g *GeneratedFile) Unskip() { g.skip = false }

// Write implements io.Writer.
func (g *GeneratedFile) Write(p []byte) (int, error) { return g.buf.Write(p) }

// Content returns the formatted content.
func (g *GeneratedFile) Content() ([]byte, error) {
	if g.skip {
		return nil, nil
	}

	// Build import block
	var importBuf bytes.Buffer
	if len(g.imports) > 0 {
		importBuf.WriteString("import (\n")

		paths := make([]GoImportPath, 0, len(g.imports))
		for p := range g.imports {
			paths = append(paths, p)
		}
		sort.Slice(paths, func(i, j int) bool { return paths[i] < paths[j] })

		for _, p := range paths {
			info := g.imports[p]
			baseName := GoPackageName(path.Base(string(p)))
			if info.packageName != baseName {
				fmt.Fprintf(&importBuf, "\t%s %q\n", info.packageName, p)
			} else {
				fmt.Fprintf(&importBuf, "\t%q\n", p)
			}
		}
		importBuf.WriteString(")\n\n")
	}

	// Insert imports after package declaration
	content := g.buf.Bytes()
	var result bytes.Buffer
	lines := bytes.Split(content, []byte("\n"))
	importInserted := false

	for i, line := range lines {
		result.Write(line)
		if i < len(lines)-1 {
			result.WriteByte('\n')
		}
		if !importInserted && bytes.HasPrefix(bytes.TrimSpace(line), []byte("package ")) {
			result.WriteByte('\n')
			result.Write(importBuf.Bytes())
			importInserted = true
		}
	}

	formatted, err := format.Source(result.Bytes())
	if err != nil {
		return result.Bytes(), fmt.Errorf("format: %w\n%s", err, result.Bytes())
	}
	return formatted, nil
}

// GoImportPath is a Go import path.
type GoImportPath string

// Ident returns a GoIdent for the given name in this import path.
func (p GoImportPath) Ident(name string) GoIdent {
	return GoIdent{GoImportPath: p, GoName: name}
}

// GoPackageName is a Go package name.
type GoPackageName string

// GoIdent is a Go identifier with its import path.
type GoIdent struct {
	GoImportPath GoImportPath
	GoName       string
}

func (id GoIdent) String() string {
	if id.GoImportPath == "" {
		return id.GoName
	}
	return string(id.GoImportPath) + "." + id.GoName
}

// GoDoc represents a documentation comment.
type GoDoc string

func (d GoDoc) PrintTo(g *GeneratedFile) {
	if d == "" {
		return
	}
	for _, line := range strings.Split(string(d), "\n") {
		g.buf.WriteString("// ")
		g.buf.WriteString(line)
		g.buf.WriteByte('\n')
	}
}

// GoParam represents a function/method parameter or return value.
type GoParam struct {
	Name string // parameter name (can be empty for returns)
	Type any    // type: string, GoIdent
}

// GoParams represents a parameter list.
type GoParams struct {
	List     []GoParam
	Variadic bool
}

func (p GoParams) PrintTo(g *GeneratedFile) {
	g.buf.WriteByte('(')
	for i, param := range p.List {
		if i > 0 {
			g.buf.WriteString(", ")
		}
		if param.Name != "" {
			g.buf.WriteString(param.Name)
			g.buf.WriteByte(' ')
		}
		if p.Variadic && i == len(p.List)-1 {
			g.buf.WriteString("...")
		}
		g.print(param.Type)
	}
	g.buf.WriteByte(')')
}

// GoResults represents a return value list.
type GoResults []GoParam

func (r GoResults) PrintTo(g *GeneratedFile) {
	if len(r) == 0 {
		return
	}
	g.buf.WriteByte(' ')
	if len(r) == 1 && r[0].Name == "" {
		g.print(r[0].Type)
		return
	}
	g.buf.WriteByte('(')
	for i, res := range r {
		if i > 0 {
			g.buf.WriteString(", ")
		}
		if res.Name != "" {
			g.buf.WriteString(res.Name)
			g.buf.WriteByte(' ')
		}
		g.print(res.Type)
	}
	g.buf.WriteByte(')')
}

// GoReceiver represents a method receiver.
type GoReceiver struct {
	Name    string // receiver name (e.g., "x")
	Type    any    // receiver type: string, GoIdent
	Pointer bool   // whether receiver is pointer
}

// GoMethod represents a method signature for code generation.
type GoMethod struct {
	Doc     GoDoc      // documentation comment (without //)
	Recv    GoReceiver // receiver
	Name    string     // method name
	Params  GoParams   // parameters
	Results GoResults  // return values
}

func (m GoMethod) PrintTo(g *GeneratedFile) {
	m.Doc.PrintTo(g)
	g.buf.WriteString("func (")
	g.buf.WriteString(m.Recv.Name)
	g.buf.WriteByte(' ')
	if m.Recv.Pointer {
		g.buf.WriteByte('*')
	}
	g.print(m.Recv.Type)
	g.buf.WriteString(") ")
	g.buf.WriteString(m.Name)
	m.Params.PrintTo(g)
	m.Results.PrintTo(g)
}

// GoFunc represents a function signature (no receiver).
type GoFunc struct {
	Doc     GoDoc
	Name    string
	Params  GoParams
	Results GoResults
}

func (f GoFunc) PrintTo(g *GeneratedFile) {
	f.Doc.PrintTo(g)
	g.buf.WriteString("func ")
	g.buf.WriteString(f.Name)
	f.Params.PrintTo(g)
	f.Results.PrintTo(g)
}

// Package represents a loaded Go package.
type Package struct {
	Name      string
	PkgPath   string
	Dir       string
	GoFiles   []string
	Fset      *token.FileSet
	TypesPkg  *types.Package
	TypesInfo *types.Info
	Syntax    []*ast.File
	Types     []*Type
	Enums     []*Enum
}

// GoImportPath returns the import path for this package.
func (p *Package) GoImportPath() GoImportPath {
	return GoImportPath(p.PkgPath)
}

// Type represents a Go type declaration.
type Type struct {
	Name     string
	Doc      string
	Pkg      *Package
	Fields   []*Field
	TypeSpec *ast.TypeSpec
}

// GoIdent returns the GoIdent for this type.
func (t *Type) GoIdent() GoIdent {
	return GoIdent{GoImportPath: t.Pkg.GoImportPath(), GoName: t.Name}
}

// Field represents a struct field.
type Field struct {
	Name    string
	Type    string
	Tag     string
	Doc     string
	Comment string
	Pos     token.Position // source position
}

// Enum represents a Go enum (type with const values).
type Enum struct {
	Name   string
	Doc    string
	Pkg    *Package
	Values []*EnumValue
}

// GoIdent returns the GoIdent for this enum.
func (e *Enum) GoIdent() GoIdent {
	return GoIdent{GoImportPath: e.Pkg.GoImportPath(), GoName: e.Name}
}

// EnumValue represents an enum constant.
type EnumValue struct {
	Name    string
	Value   string
	Doc     string
	Comment string
	Pos     token.Position // source position
}

// Helper functions

func buildFlags(tags []string) []string {
	if len(tags) == 0 {
		return nil
	}
	return []string{"-tags=" + strings.Join(tags, ",")}
}

func pkgDir(pkg *packages.Package) string {
	if len(pkg.GoFiles) > 0 {
		return filepath.Dir(pkg.GoFiles[0])
	}
	return ""
}

func docText(cg *ast.CommentGroup) string {
	if cg == nil {
		return ""
	}
	return cg.Text()
}

func commentText(cg *ast.CommentGroup) string {
	if cg == nil {
		return ""
	}
	return strings.TrimSpace(cg.Text())
}

func exprString(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		return exprString(e.X) + "." + e.Sel.Name
	case *ast.StarExpr:
		return "*" + exprString(e.X)
	case *ast.ArrayType:
		if e.Len == nil {
			return "[]" + exprString(e.Elt)
		}
		return fmt.Sprintf("[%s]%s", exprString(e.Len), exprString(e.Elt))
	case *ast.MapType:
		return fmt.Sprintf("map[%s]%s", exprString(e.Key), exprString(e.Value))
	case *ast.BasicLit:
		return e.Value
	default:
		return fmt.Sprintf("%T", expr)
	}
}

func extractFields(fset *token.FileSet, st *ast.StructType) []*Field {
	var fields []*Field
	for _, f := range st.Fields.List {
		for _, name := range f.Names {
			field := &Field{
				Name:    name.Name,
				Type:    exprString(f.Type),
				Doc:     docText(f.Doc),
				Comment: commentText(f.Comment),
				Pos:     fset.Position(name.Pos()),
			}
			if f.Tag != nil {
				field.Tag = f.Tag.Value
			}
			fields = append(fields, field)
		}
	}
	return fields
}

// OutputPath joins directory and filename.
func OutputPath(dir, filename string) string {
	return filepath.Join(dir, filename)
}

// RawString represents a raw string literal (backtick-quoted) for code generation.
// Use this for regex patterns or other strings that should not be escaped.
type RawString string

func (r RawString) PrintTo(g *GeneratedFile) {
	g.buf.WriteByte('`')
	g.buf.WriteString(string(r))
	g.buf.WriteByte('`')
}
