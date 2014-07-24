package main

import (
	"errors"

	"gopkgs.com/browser.v1"
	"gopkgs.com/cmd/gopkgs/lib"
)

func viewCommand(args []string) error {
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
	return browser.Open(repo.GoPkgsPath)
}
