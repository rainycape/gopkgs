package main

import (
	"errors"

	"gopkgs.com/browser.v1"
	"gopkgs.com/cmd/gopkgs/lib"
)

type docOptions struct {
	PreferRevisions bool `name:"r" help:"Prefer revisions to versions"`
}

func docCommand(args []string, opts *docOptions) error {
	if len(args) == 0 {
		return errors.New("missing package import path")
	}
	req := &lib.RepoRequest{
		Path: args[0],
	}
	repo, err := Repo(req)
	if err != nil {
		return err
	}
	var url string
	if opts.PreferRevisions {
		url = repo.RevisionDocumentation()
	} else {
		url = repo.VersionDocumentation()
	}
	return browser.Open(url)
}
