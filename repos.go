package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"gopkgs.com/cmd/gopkgs/lib"
)

const (
	apiHost    = "gopkgs.com"
	apiVersion = "1"
)

func getApiHost() string {
	if host := os.Getenv("GOPKGS_API_HOST"); host != "" {
		return host
	}
	return apiHost
}

func apiPath(p string) string {
	return "http://" + getApiHost() + "/api/v" + apiVersion + p
}

func Repos(reqs []*lib.RepoRequest) ([]*lib.Repo, error) {
	postData, err := json.Marshal(reqs)
	if err != nil {
		return nil, err
	}
	resp, err := http.Post(apiPath("/info"), "application/json", bytes.NewReader(postData))
	if err != nil {
		return nil, err

	}
	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%d status code from %s: %s", resp.StatusCode, getApiHost(), string(data))
	}
	var repos []*lib.Repo
	if err := json.Unmarshal(data, &repos); err != nil {
		return nil, fmt.Errorf("error decoding JSON: %s\nResponse:\n%s\n", err, string(data))
	}
	return repos, nil
}

func Repo(req *lib.RepoRequest) (*lib.Repo, error) {
	repos, err := Repos([]*lib.RepoRequest{req})
	if err != nil {
		return nil, err
	}
	if repos[0].Error != "" {
		return nil, errors.New(repos[0].Error)
	}
	return repos[0], nil
}
