// This file is part of GoRE.
//
// Copyright (C) 2019-2021 GoRE Authors
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

//go:build ignore
// +build ignore

// This program generates stdpkgs_gen.go. It can be invoked by running
// go generate

package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"go/format"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

var packageTemplate = template.Must(template.New("").Parse(`// This file is part of GoRE.
//
// Copyright (C) 2019-2021 GoRE Authors
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

// Code generated by go generate; DO NOT EDIT.
// This file was generated at
// {{ .Timestamp }}

package gore

var stdPkgs = map[string]struct{}{
{{- range .StdPkg }}
	{{ printf "\"%s\": {}" . }},
{{- end }}
}
`))

var goversionTemplate = template.Must(template.New("").Parse(`// This file is part of GoRE.
//
// Copyright (C) 2019-2021 GoRE Authors
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

// Code generated by go generate; DO NOT EDIT.
// This file was generated at
// {{ .Timestamp }}

package gore

var goversions = map[string]*GoVersion{
{{- range .GoVersions }}
	{{ printf "\"%s\": {Name: \"%s\", SHA: \"%s\", Timestamp: \"%s\"}" .Name .Name .Sha .Date }},
{{- end }}
}
`))

var client = &http.Client{}

var authRequest func(*http.Request)

func init() {
	token := os.Getenv("GITHUB_TOKEN")
	if token != "" {
		authRequest = func(r *http.Request) {
			r.Header.Set("Authorization", "token "+token)
		}
	} else {
		authRequest = func(r *http.Request) {}
	}
}

type ghResp struct {
	Sha       string   `json:"sha"`
	Url       string   `json:"url"`
	Trees     []ghTree `json:"tree"`
	Truncated bool     `json:"truncated"`
}

type ghTree struct {
	Path    string `json:"path"`
	Mode    string `json:"mode"`
	Gittype string `json:"type"`
	Sha     string `json:"sha"`
	Size    int    `json:"size"`
	Url     string `json:"url"`
}

const (
	requestURLFormatStr       = "https://api.github.com/repos/golang/go/git/trees/%s?recursive=0"
	commitRequestURLFormatStr = "https://api.github.com/repos/golang/go/git/commits/%s"
	outputFile                = "stdpkg_gen.go"
	goversionOutputFile       = "goversion_gen.go"
)

var (
	tagsRequestURL = "https://api.github.com/repos/golang/go/tags"
)

var (
	excludedPaths = []string{"src/cmd"}
)

type tagResp struct {
	Name   string
	Commit *commitShort
}

type commitShort struct {
	Sha string
	URL string
}

type commitLong struct {
	Sha       string
	Committer committer
}

type committer struct {
	Name string
	Date string
}

type goversion struct {
	Name string
	Sha  string
	Date string
}

// diffCode returns false if a and b have different other than the date.
func diffCode(a, b string) bool {
	if a == b {
		return false
	}

	aLines := strings.Split(a, "\n")
	bLines := strings.Split(b, "\n")

	// ignore the license and the date
	aLines = aLines[21:]
	bLines = bLines[21:]

	if len(aLines) != len(bLines) {
		return true
	}

	for i := 0; i < len(aLines); i++ {
		if aLines[i] != bLines[i] {
			return true
		}
	}

	return false
}

func writeOnDemand(new []byte, target string) {
	old, err := os.ReadFile(target)
	if err != nil {
		fmt.Println("Error when reading the old file:", target, err)
		return
	}

	old, _ = format.Source(old)
	new, _ = format.Source(new)

	// Compare the old and the new.
	if !diffCode(string(old), string(new)) {
		fmt.Println(target + " no changes.")
		return
	}

	fmt.Println(target + " changes detected.")

	// Write the new file.
	err = os.WriteFile(target, new, 0664)
	if err != nil {
		fmt.Println("Error when writing the new file:", err)
		return
	}
}

func processGoVersions() {
	tags := make([]*tagResp, 0)

	// Fetch all tags

	var requestURL *string

	requestURL = &tagsRequestURL
	for *requestURL != "" {
		fmt.Println("Fetching latests tags")
		req, _ := http.NewRequest(http.MethodGet, *requestURL, nil)
		authRequest(req)
		resp, err := client.Do(req)
		if err != nil {
			fmt.Println("Error when fetching tags:", err.Error())
			resp.Body.Close()
			continue
		}
		next := getNextPageURL(resp)
		*requestURL = next
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			fmt.Println("Error when ready response body:", err)
			continue
		}
		var newTags []*tagResp
		err = json.Unmarshal(body, &newTags)
		if err != nil {
			fmt.Println("Error when parsing the json:", string(body), err)
			continue
		}
		tags = append(tags, newTags...)
	}

	// Get mode commit info for new tags

	f, err := os.OpenFile(filepath.Join("resources", "goversions.csv"), os.O_CREATE|os.O_RDWR, 0664)
	if err != nil {
		fmt.Println("Error when opening goversions.csv:", err)
		return
	}
	defer f.Close()
	knownVersions, err := getStoredGoversions(f)
	if err != nil {
		fmt.Println("Error when getting stored go versions:", err)
		return
	}

	_, err = fmt.Fprintln(f, "version,sha,date")
	if err != nil {
		fmt.Println("Error when writing csv header:", err)
		return
	}

	for _, tag := range tags {
		if strings.HasPrefix(tag.Name, "weekly") || strings.HasPrefix(tag.Name, "release") {
			continue
		}
		if v, known := knownVersions[tag.Name]; known {
			fmt.Fprintf(f, "%s,%s,%s\n", v.Name, v.Sha, v.Date)
			continue
		}

		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf(commitRequestURLFormatStr, tag.Commit.Sha), nil)
		authRequest(req)
		resp, err := client.Do(req)
		if err != nil {
			fmt.Println("Error when fetching commit info:", err)
			resp.Body.Close()
			continue
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()

		var commit commitLong
		err = json.Unmarshal(body, &commit)
		if err != nil {
			fmt.Println("Error when parsing commit json:", err)
			continue
		}
		fmt.Fprintf(f, "%s,%s,%s\n", tag.Name, commit.Sha, commit.Committer.Date)
		fmt.Println("New tag found:", tag.Name)
		knownVersions[tag.Name] = &goversion{Name: tag.Name, Sha: commit.Sha, Date: commit.Committer.Date}
	}

	// Generate the code.
	buf := bytes.NewBuffer(nil)

	err = goversionTemplate.Execute(buf, struct {
		Timestamp  time.Time
		GoVersions map[string]*goversion
	}{
		Timestamp:  time.Now().UTC(),
		GoVersions: knownVersions,
	})
	if err != nil {
		fmt.Println("Error when generating the code:", err)
		return
	}

	writeOnDemand(buf.Bytes(), goversionOutputFile)
}

func getStoredGoversions(f *os.File) (map[string]*goversion, error) {
	vers := make(map[string]*goversion)
	r := bufio.NewScanner(f)
	// Read header
	if !r.Scan() {
		return nil, errors.New("empty file")
	}
	r.Text()

	for r.Scan() {
		row := r.Text()
		if row == "" {
			continue
		}
		data := strings.Split(row, ",")
		if data[0] == "" {
			// No version
			continue
		}
		version := strings.TrimSpace(data[0])
		sha := strings.TrimSpace(data[1])
		date := strings.TrimSpace(data[2])
		vers[version] = &goversion{Name: version, Sha: sha, Date: date}
	}
	_, err := f.Seek(0, 0)
	return vers, err
}

func getNextPageURL(r *http.Response) string {
	h := r.Header.Get("Link")
	if h == "" {
		return ""
	}
	// Either we this type:
	// <https://api.github.com/repositories/23096959/tags?page=2>; rel="next", <https://api.github.com/repositories/23096959/tags?page=8>; rel="last"
	// or this type:
	// <https://api.github.com/repositories/23096959/tags?page=7>; rel="prev", <https://api.github.com/repositories/23096959/tags?page=1>; rel="first"
	data := strings.Split(h, ",")
	for _, l := range data {
		ll := strings.Split(l, ";")
		if len(ll) != 2 {
			continue
		}
		if strings.TrimSpace(ll[1]) != "rel=\"next\"" {
			continue
		}
		return strings.TrimLeft(strings.TrimRight(strings.TrimSpace(ll[0]), ">"), "<")
	}
	return ""
}

func main() {
	processGoVersions()

	resp, err := client.Get(fmt.Sprintf(requestURLFormatStr, "master"))
	if err != nil {
		fmt.Println("Error when fetching go src data:", err)
		return
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error when reading response body:", err)
		return
	}
	var master ghResp
	err = json.Unmarshal(body, &master)
	if err != nil {
		fmt.Println("Error when decoding the response body:", err)
		return
	}
	var stdPkgs []string
	for _, tree := range master.Trees {
		if tree.Gittype != "tree" {
			continue
		}
		if !strings.HasPrefix(tree.Path, "src") || skipPath(tree.Path) {
			continue
		}
		// Skip src folder.
		if tree.Path == "src" {
			continue
		}
		// Strip "src/" and add to the list.
		stdPkgs = append(stdPkgs, strings.TrimPrefix(tree.Path, "src/"))
	}

	// Generate the code.
	buf := bytes.NewBuffer(nil)

	err = packageTemplate.Execute(buf, struct {
		Timestamp time.Time
		StdPkg    []string
	}{
		Timestamp: time.Now().UTC(),
		StdPkg:    stdPkgs,
	})
	if err != nil {
		fmt.Println("Error when generating the code:", err)
		return
	}

	writeOnDemand(buf.Bytes(), outputFile)
}

func skipPath(path string) bool {
	for _, exclude := range excludedPaths {
		if strings.HasPrefix(path, exclude) {
			return true
		}
	}
	return false
}
