package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/build"
	"go/format"
	"go/parser"
	"go/printer"
	"go/token"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"gopkgs.com/cmd/gopkgs/lib"

	"code.google.com/p/go.crypto/ssh/terminal"
	"code.google.com/p/go.tools/astutil"
)

type autoBool string

func (b *autoBool) Auto() bool {
	return strings.ToLower(b.String()) == "auto"
}

func (b *autoBool) Bool() bool {
	if b.Auto() {
		return false
	}
	v, _ := strconv.ParseBool(b.String())
	return v
}

func (b *autoBool) String() string {
	return string(*b)
}

func (b *autoBool) Set(s string) error {
	if _, err := strconv.ParseBool(s); err != nil {
		return err
	}
	*b = autoBool(s)
	return nil
}

func (b *autoBool) IsBoolFlag() bool {
	return true
}

func (b *autoBool) compileTimeCheck() flag.Value {
	return b
}

type rewriteOptions struct {
	Interactive     bool     `name:"i" help:"Interactive mode"`
	PreferRevisions bool     `name:"r" help:"Prefer revisions to versions"`
	Library         autoBool `name:"lib" help:"[auto|true|false]: Library mode - refuse to pin packages on revisions, only on versions"`
	DryRun          bool     `name:"n" help:"Dry run - only show the changes that would be made"`
	Verbose         bool     `name:"v" help:"Verbose output"`
}

func (opts *rewriteOptions) LibraryMode(pkg *build.Package) bool {
	if opts.Library.Auto() {
		return pkg.Name != "main"
	}
	return opts.Library.Bool()
}

type rewriteState struct {
	repos          map[string]*lib.Repo
	downloadErrors map[string]error
}

func (r *rewriteState) key(req *lib.RepoRequest) string {
	return req.Path + "|" + req.Revision
}

func (r *rewriteState) Repos(reqs []*lib.RepoRequest) ([]*lib.Repo, error) {
	var pending []*lib.RepoRequest
	for _, v := range reqs {
		if r.repos[r.key(v)] == nil {
			pending = append(pending, v)
		}
	}
	if len(pending) > 0 {
		resp, err := Repos(pending)
		if err != nil {
			return nil, err
		}
		if r.repos == nil {
			r.repos = make(map[string]*lib.Repo)
		}
		for ii, v := range resp {
			r.repos[r.key(pending[ii])] = v
		}
	}
	ret := make([]*lib.Repo, len(reqs))
	for ii, v := range reqs {
		ret[ii] = r.repos[r.key(v)]
	}
	return ret, nil
}

func (r *rewriteState) RequestRepos(names []string) ([]*lib.Repo, error) {
	reqs := make([]*lib.RepoRequest, len(names))
	for ii, v := range names {
		reqs[ii] = &lib.RepoRequest{
			Path: v,
		}
	}
	return Repos(reqs)
}

func (r *rewriteState) DownloadImport(p string, opts *rewriteOptions) error {
	if err, ok := r.downloadErrors[p]; ok {
		return err
	}
	var err error
	if _, ierr := build.Import(p, "", 0); ierr != nil {
		args := []string{"get"}
		if opts.Verbose {
			args = append(args, "-v")
		}
		args = append(args, p)
		cmd := exec.Command("go", args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Run()
	}
	if r.downloadErrors == nil {
		r.downloadErrors = make(map[string]error)
	}
	r.downloadErrors[p] = err
	return err
}

func pkgName(pkg *build.Package) string {
	if pkg.Name != "main" {
		return pkg.Name
	}
	return pkg.Name
}

func parseFiles(fset *token.FileSet, abspath string, names []string, mode parser.Mode) (map[string]*ast.File, error) {
	files := make(map[string]*ast.File)
	for _, f := range names {
		absname := filepath.Join(abspath, f)
		file, err := parser.ParseFile(fset, absname, nil, mode)
		if err != nil {
			// Just ignore this file
			continue
		}
		files[absname] = file
	}
	return files, nil
}

func rewritePackage(pkg *build.Package, st *rewriteState, opts *rewriteOptions) {
	if err := doRewritePackage(pkg, st, opts); err != nil {
		log.Printf("error rewriting package %s: %s", pkgName(pkg), err)
	}
}

func rewriteImports(fset *token.FileSet, pkg *build.Package, files map[string]*ast.File, rewrites map[string]string, st *rewriteState, opts *rewriteOptions) error {
	for k, v := range files {
		rewritten := make(map[string]string)
		imports := astutil.Imports(fset, v)
		for _, group := range imports {
			for _, imp := range group {
				if unquoted, err := strconv.Unquote(imp.Path.Value); err == nil {
					for rk, rv := range rewrites {
						if !strings.HasPrefix(unquoted, rk) {
							continue
						}
						newImport := strings.Replace(unquoted, rk, rv, 1)
						if !opts.DryRun {
							if err := st.DownloadImport(newImport, opts); err != nil {
								fmt.Fprintf(os.Stderr, "couldn't download %s, using original", newImport)
								continue
							}
						}
						rewritten[unquoted] = newImport
					}
				}
			}
		}
		if len(rewritten) == 0 {
			continue
		}
		if opts.DryRun || opts.Verbose {
			if opts.DryRun {
				fmt.Printf("would rewrite %d imports in %s:\n", len(rewritten), k)
			} else {
				fmt.Printf("rewrite %d imports in %s:\n", len(rewritten), k)
			}
			for ik, iv := range rewritten {
				fmt.Printf("\t%s => %s\n", ik, iv)
			}
			if opts.DryRun {
				continue
			}
		}
		for ik, iv := range rewritten {
			astutil.RewriteImport(fset, v, ik, iv)
		}
		// Same as go fmt
		cfg := &printer.Config{
			Tabwidth: 8,
			Mode:     printer.UseSpaces | printer.TabIndent,
		}
		var buf bytes.Buffer
		var data []byte
		var st os.FileInfo
		var err error
		if err = cfg.Fprint(&buf, fset, v); err == nil {
			if data, err = format.Source(buf.Bytes()); err == nil {
				if st, err = os.Stat(k); err == nil {
					err = ioutil.WriteFile(k, data, st.Mode())
				}
			}
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "error rewriting file %s: %s\n", k, err)
		}

	}
	return nil
}

func pkgFromExpr(expr ast.Expr) string {
	switch x := expr.(type) {
	case *ast.SelectorExpr:
		if ident, ok := x.X.(*ast.Ident); ok {
			return ident.Name
		}
	case *ast.StarExpr:
		return pkgFromExpr(x.X)
	}
	return ""
}

func shouldKeepOriginalImport(fset *token.FileSet, p string, spec *ast.ImportSpec, file *ast.File, opts *rewriteOptions) bool {
	if strings.HasPrefix(p, lib.GoPkgsPrefix) {
		return true
	}
	// If the file contains any type assertions on any type
	// from the package, don't suggest to rewrite it, since it
	// might be asserting on an interface which came from another
	// 3rd party library and the imports in the 3rd party library and
	// the current code must match, otherwise the assertion will fail
	// at runtime, while still compiling fine.
	var name string
	if spec.Name != nil {
		name = spec.Name.Name
	}
	if name == "" {
		// Import the package and check its name
		pkg, err := build.Import(p, "", 0)
		if err != nil {
			// Can't find original package, keep it
			fmt.Fprintf(os.Stderr, "can't find import %s: %s", p, err)
			return true
		}
		name = pkg.Name
	}
	if name == "" || name == "." {
		return true
	}
	keep := false
	ast.Inspect(file, func(n ast.Node) bool {
		if ta, ok := n.(*ast.TypeAssertExpr); ok && name == pkgFromExpr(ta.Type) {
			if opts.Verbose {
				pos := fset.Position(n.Pos())
				fmt.Printf("keeping import %s due to type assertion in %s:%d\n", p, pos.Filename, pos.Line)
			}
			keep = true
		}
		if cc, ok := n.(*ast.CaseClause); ok {
			for _, expr := range cc.List {
				if name == pkgFromExpr(expr) {
					if opts.Verbose {
						pos := fset.Position(n.Pos())
						fmt.Printf("keeping import %s due to case in type switch in %s:%d\n", p, pos.Filename, pos.Line)
					}
					keep = true
				}
			}
		}
		// if keep is already true, we can stop iterating the AST
		return !keep
	})
	return keep
}

func doRewritePackage(pkg *build.Package, st *rewriteState, opts *rewriteOptions) error {
	libraryMode := opts.LibraryMode(pkg)
	abs, err := filepath.Abs(pkg.Dir)
	if err != nil {
		return err
	}
	fset := token.NewFileSet()
	var names []string
	names = append(names, pkg.GoFiles...)
	names = append(names, pkg.CgoFiles...)
	files, err := parseFiles(fset, abs, names, parser.ParseComments)
	if err != nil {
		return err
	}
	// First check if we should keep any original imports in the package due to
	// the use it makes of the imported pkg (type assertions, etc...).
	disabled := make(map[string]bool)
	for _, v := range files {
		imports := astutil.Imports(fset, v)
		for _, group := range imports {
			for _, imp := range group {
				if unquoted, err := strconv.Unquote(imp.Path.Value); err == nil {
					m := repositoryRe.FindStringSubmatch(unquoted)
					if len(m) > 0 && shouldKeepOriginalImport(fset, unquoted, imp, v, opts) {
						disabled[unquoted] = true
					}
				}
			}
		}
	}
	using := make(map[string]bool)
	for _, v := range files {
		imports := astutil.Imports(fset, v)
		for _, group := range imports {
			for _, imp := range group {
				if unquoted, err := strconv.Unquote(imp.Path.Value); err == nil {
					m := repositoryRe.FindStringSubmatch(unquoted)
					if len(m) > 0 && !disabled[unquoted] {
						using[m[0]] = true
					}
				}
			}
		}
	}
	// Now check imports we should rewrite
	if len(using) == 0 {
		return nil
	}
	var repoNames []string
	for k := range using {
		repoNames = append(repoNames, k)
	}
	if opts.Verbose {
		fmt.Printf("package %s uses %d 3rd party repositories: %v\n", pkgName(pkg), len(repoNames), repoNames)
	}
	repos, err := st.RequestRepos(repoNames)
	if err != nil {
		return err
	}
	rewrites := make(map[string]string)
addRewrites:
	for _, v := range repos {
		var importPath string
		if libraryMode {
			if v.Version == 0 {
				if v.AllowsUnpinned {
					importPath = v.GoPkgsPath
				} else {
					if opts.Verbose {
						fmt.Printf("ignoring package %s, no versions available\n", v.Path)
					}
					continue
				}
			} else {
				importPath = v.VersionImportPath()
			}
		} else {
			if opts.PreferRevisions {
				importPath = v.RevisionImportPath()
			} else {
				importPath = v.VersionImportPath()
			}
		}
		if opts.Interactive {
		prompt:
			for {
				fmt.Printf("rewrite import %s to %s in package %s? (y/N)", v.Path, importPath, pkgName(pkg))
				oldState, err := terminal.MakeRaw(0)
				if err != nil {
					panic(err)
				}
				var buf [1]byte
				os.Stdin.Read(buf[:])
				terminal.Restore(0, oldState)
				fmt.Print("\n")
				switch buf[0] {
				case 'y', 'Y':
					break prompt
				case 'n', 'N', '\r': // /r is enter
					continue addRewrites
				case '\x03', '\x01':
					// ctrl+c, ctrl+z
					os.Exit(0)
				}
			}

		}
		rewrites[v.Path] = importPath
	}
	if len(rewrites) == 0 {
		return nil
	}
	// TODO go get new imports
	return rewriteImports(fset, pkg, files, rewrites, st, opts)
}

func rewriteSubcommand(args []string, opts *rewriteOptions) {
	st := new(rewriteState)
	if len(args) > 0 {
		for _, v := range args {
			if pkg, err := build.ImportDir(v, 0); err == nil {
				rewritePackage(pkg, st, opts)
				continue
			}
			pkg, err := build.Import(v, "", 0)
			if err != nil {
				log.Printf("error importing %s: %s", v, err)
				continue
			}
			rewritePackage(pkg, st, opts)
		}
	} else {
		abs, err := filepath.Abs(".")
		if err != nil {
			panic(err)
		}
		pkg, err := build.ImportDir(abs, 0)
		if err != nil {
			log.Fatalf("error importing %s: %s", abs, err)
		}
		rewritePackage(pkg, st, opts)
	}
}
