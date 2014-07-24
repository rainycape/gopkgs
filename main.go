package main

import (
	"regexp"

	"gopkgs.com/cmd/gopkgs/lib"
	"gopkgs.com/command.v1"
)

var (
	repositoryRe = regexp.MustCompile("^(?:" + lib.GitHubPattern + "|" + lib.GoogleCodePattern + "|" + lib.GoPkgsPattern + ")")
)

func main() {
	command.Run(commands)
}
