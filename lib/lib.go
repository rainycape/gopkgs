// Package lib contains types and constants used by both the gopkgs command and site.
package lib

import "fmt"

const (
	GitHubShortcut     = "gh"
	GoogleCodeShortcut = "gc"
	BitBucketShortcut  = "bb"

	GitHubPrefix     = `github.com/`
	GoogleCodePrefix = `code.google.com/p/`
	GoPkgsPrefix     = `gopkgs.com/`

	GoPkgsGitHubPrefix     = "gh"
	GoPkgsGoogleCodePrefix = "gc"

	GitHubPattern     = `github\.com/(?P<github_repo>[A-Za-z0-9_.\-]+/[A-Za-z0-9_.\-]+)`
	GoogleCodePattern = `code.google.com/p/(?P<google_repo>[A-Za-z0-9\-]+(?:\.[A-Za-z0-9]+)?)`

	GoPkgsPattern = `gopkgs.com/(?P<gopkgs_repo>[A-Za-z0-9]+)`
)

type RepoRequest struct {
	Path     string `json:"path"`
	Revision string `json:"revision"`
}

type Repo struct {
	Path                string `json:"path"`
	GoPkgsPath          string `json:"gopkgs_path"`
	Version             int    `json:"version"`
	Revision            string `json:"revision"`
	AllowsUnpinned      bool   `json:"allows_unpinned"`
	Error               string `json:"error"`
	DocumentationPrefix string `json:"documentation_prefix"`
}

func (r *Repo) VersionImportPath() string {
	if r.Version > 0 {
		return fmt.Sprintf("%s.v%d", r.GoPkgsPath, r.Version)
	}
	return r.RevisionImportPath()
}

func (r *Repo) RevisionImportPath() string {
	if r.Revision != "" {
		return fmt.Sprintf("%s.r%s", r.GoPkgsPath, r.Revision)
	}
	return r.GoPkgsPath
}

func (r *Repo) VersionDocumentation() string {
	return r.DocumentationPrefix + r.VersionImportPath()
}

func (r *Repo) RevisionDocumentation() string {
	return r.DocumentationPrefix + r.RevisionImportPath()
}
