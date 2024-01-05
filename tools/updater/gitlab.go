package main

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"sort"
	"time"

	"github.com/Masterminds/semver"

	"code-intelligence.com/cifuzz/pkg/log"
)

type gitlabPackage struct {
	Name    string `json:"name"`
	ID      int    `json:"id"`
	Type    string `json:"package_type"`
	Version string `json:"version"`
}

func findLatestVersion(dep string) string {
	var gitlabPackageType string
	var gitlabPackageName string
	switch dep {
	case "gradle-plugin":
		gitlabPackageType = "maven"
		gitlabPackageName = "com/code-intelligence/cifuzz-gradle-plugin"
	case "maven-extension":
		gitlabPackageType = "maven"
		gitlabPackageName = "com/code-intelligence/cifuzz-maven-extension"
	case "jazzer":
		gitlabPackageType = "maven"
		gitlabPackageName = "com/code-intelligence/jazzer-junit"
	case "jazzerjs":
		gitlabPackageType = "npm"
		gitlabPackageName = "@jazzer.js/jest-runner"
	default:
		log.ErrorMsgf("unknow dep %s", dep)
		os.Exit(1)
	}

	token, ok := os.LookupEnv("GITLAB_UPDATER_TOKEN")
	if !ok {
		log.ErrorMsgf("GITLAB_UPDATER_TOKEN not set")
		os.Exit(1)
	}

	// build request
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", "https://gitlab.code-intelligence.com/api/v4/projects/89/packages", nil)
	handleErr(err)
	req.Header.Add("PRIVATE-TOKEN", token)
	q := req.URL.Query()
	q.Add("package_name", gitlabPackageName)
	q.Add("package_type", gitlabPackageType)
	q.Add("sort", "desc")
	q.Add("per_page", "100")
	req.URL.RawQuery = q.Encode()

	resp, err := client.Do(req)
	handleErr(err)
	defer resp.Body.Close()

	// handle responbse body
	body, err := io.ReadAll(resp.Body)
	handleErr(err)

	var packages []gitlabPackage
	err = json.Unmarshal(body, &packages)
	handleErr(err)

	// find latest version
	var versions semver.Collection
	for _, pkg := range packages {
		versions = append(versions, semver.MustParse(pkg.Version))
	}
	sort.Sort(versions)
	return versions[len(versions)-1].String()
}
