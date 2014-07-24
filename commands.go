package main

import (
	"gopkgs.com/command.v1"
)

var (
	importPathHelp = `

<import-path> might be either the original package import path, like
github.com/rainycape/vfs or code.google.com/p/go.tools, or either the
import path at gopkgs.com, like gopkgs.com/vfs or gopkgs.com/gc/go.tools.`

	docHelp = `doc shows the package documentation for the given
package in the default web browser. By default, doc will initially
open the latest available version of the package. The -r flag 
might be used to open the latest revision instead.` + importPathHelp

	viewHelp = `view shows the given package at gopkgs.com in the
default web browser. This command can be used to view all the available
versions and revisions of a given package.` + importPathHelp
)

var (
	commands = []*command.Cmd{
		{
			Name:     "doc",
			Help:     "Open package documentation in the default browser",
			LongHelp: docHelp,
			Usage:    "<import-path>",
			Func:     docCommand,
			Options:  &docOptions{},
		},
		{
			Name:    "get",
			Help:    "Download or update go packages from gopkgs.com",
			Func:    getCommand,
			Options: &getOptions{},
		},
		{
			Name:     "view",
			Help:     "View package at gopkgs.com",
			LongHelp: viewHelp,
			Usage:    "<import-path>",
			Func:     viewCommand,
			Options:  nil,
		},
	}
)
