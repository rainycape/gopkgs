package main

import (
	"gopkgs.com/command.v1"
)

var (
	rewriteHelp = `rewrite rewrites import paths in the given packages to use
gopkgs.com import paths when possible.

Packages might be specified either as import paths, or file paths to package
directories (either absolute or relative). If no packages are specified, the package
at the current directory is used.

By default, rewrite will try to use package versions, falling back to package revisions
when the package has no available versions. This behavior can be changed using the -r
flag.

Users should generally avoid pinning packages on exact revisions when writing reusable
libraries. For this reason, the -lib flag defaults to auto, which will enable library mode
when the package name is different than "main". When enabled, this flag causes rewrite to
refuse pinning packages on revisions, taking precedence over the -r flag. This behavior
might be overridden by setting either -lib=true or -lib=false, but is usually not
recommended to do so.`
	importPathHelp = `

<import-path> might be either the original package import path, like
github.com/rainycape/vfs or code.google.com/p/go.tools, or either the
import path at gopkgs.com, like gopkgs.com/vfs or gopkgs.com/gc/go.tools.`

	docHelp = `doc shows the package documentation for the given
package in the default web browser. By default, doc will initially
open the latest available version of the package. The -r flag 
might be used to open the latest revision instead.` + importPathHelp

	getHelp = `get downloads packages using gopkgs.com import paths.
By default, get will download the latest available version of the package.
The -r flag might be used to download the latest revision instead.` + importPathHelp

	viewHelp = `view shows the given package at gopkgs.com in the
default web browser. This command can be used to view all the available
versions and revisions of a given package.` + importPathHelp
)

var (
	commands = []*command.Cmd{
		{
			Name:     "rewrite",
			Help:     "Rewrite import paths to use gopkgs.com when possible",
			LongHelp: rewriteHelp,
			Usage:    "[pkg-1] [pkg-2] ... [pkg-n]",
			Func:     rewriteSubcommand,
			Options:  &rewriteOptions{Library: "auto"},
		},
		{
			Name:     "doc",
			Help:     "Open package documentation in the default browser",
			LongHelp: docHelp,
			Usage:    "<import-path>",
			Func:     docCommand,
			Options:  &docOptions{},
		},
		{
			Name:     "get",
			Help:     "Download or update go packages from gopkgs.com",
			Usage:    "<import-path-1> [import-path-2] ... [import-path-n]",
			LongHelp: getHelp,
			Func:     getCommand,
			Options:  &getOptions{},
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
