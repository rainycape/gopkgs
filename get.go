package main

import (
	"errors"
	"fmt"
	"go/build"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkgs.com/cmd/gopkgs/lib"
)

type getOptions struct {
	Update          bool `name:"u" help:"If a package is already downloaded, update it and its dependencies"`
	PreferRevisions bool `name:"r" help:"Prefer revisions to versions"`
	Verbose         bool `name:"v" help:"Verbose output"`
}

func runGoGet(r *lib.Repo, opts *getOptions) error {
	args := []string{"get"}
	if opts.Update {
		args = append(args, "-u")
	}
	if opts.Verbose {
		args = append(args, "-v")
	}
	var importPath string
	if strings.HasPrefix(r.Path, lib.GoPkgsPrefix) || r.GoPkgsPath == "" {
		// Package either was initially specified as a gopkgs.com import path,
		// or unknown gopkgs.com
		if r.Error != "" {
			fmt.Printf("gopkgs can't find package %s: %s - using original\n", r.Path, r.Error)
		}
		importPath = r.Path
	} else {
		if opts.PreferRevisions {
			importPath = r.RevisionImportPath()
		} else {
			importPath = r.VersionImportPath()
		}
		fmt.Printf("using %s for package %s\n", importPath, r.Path)
	}
	args = append(args, importPath)
	cmd := exec.Command("go", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func listGoPkgsPackages() ([]string, error) {
	var pkgs []string
	for _, goPath := range filepath.SplitList(build.Default.GOPATH) {
		abs, err := filepath.Abs(filepath.Join(goPath, "src"))
		if err != nil {
			continue
		}
		trim := abs + string(filepath.Separator)
		filepath.Walk(filepath.Join(abs, lib.GoPkgsPrefix), func(s string, info os.FileInfo, err error) error {
			if err != nil || !info.IsDir() {
				return err
			}
			dotGit := filepath.Join(s, ".git")
			if st, err := os.Stat(dotGit); err == nil && st.IsDir() {
				pkgs = append(pkgs, strings.TrimPrefix(s, trim))
			}
			return nil
		})
	}
	return pkgs, nil
}

func getCommand(args []string, opts *getOptions) error {
	if len(args) == 0 {
		if opts.Update {
			// Gather all packages from gopkgs.com
			args, _ = listGoPkgsPackages()
		}
		if len(args) == 0 {
			return errors.New("no packages specified")
		}
	}
	var reqs []*lib.RepoRequest
	for _, v := range args {
		reqs = append(reqs, &lib.RepoRequest{Path: v})
	}
	repos, err := Repos(reqs)
	if err != nil {
		return err
	}
	for _, r := range repos {
		runGoGet(r, opts)
	}
	return nil
}
