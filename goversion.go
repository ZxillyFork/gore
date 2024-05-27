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

//go:generate go run ./gen goversion

package gore

import (
	"bytes"
	"errors"
	"regexp"

	"github.com/ZxillyFork/gore/extern"
	"github.com/ZxillyFork/gore/extern/gover"
)

var goVersionMatcher = regexp.MustCompile(`(go[\d+.]*(beta|rc)?[\d*])`)

// GoVersion holds information about the compiler version.
type GoVersion struct {
	// Name is a string representation of the version.
	Name string
	// SHA is a digest of the git commit for the release.
	SHA string
	// Timestamp is a string of the timestamp when the commit was created.
	Timestamp string
}

// ResolveGoVersion tries to return the GoVersion for the given tag.
// For example the tag: go1 will return a GoVersion struct representing version 1.0 of the compiler.
// If no goversion for the given tag is found, nil is returned.
func ResolveGoVersion(tag string) *GoVersion {
	v, ok := goversions[tag]
	if !ok {
		return nil
	}
	return v
}

// GoVersionCompare compares two version strings.
// If a < b, -1 is returned.
// If a == b, 0 is returned.
// If a > b, 1 is returned.
func GoVersionCompare(a, b string) int {
	if a == b {
		return 0
	}
	a = extern.StripGo(a)
	b = extern.StripGo(b)
	return gover.Compare(a, b)
}

func findGoCompilerVersion(f *GoFile) (*GoVersion, error) {
	// if DWARF debug info exists, then this can simply be obtained from there
	if gover, ok := getBuildVersionFromDwarf(f.fh); ok {
		if ver := ResolveGoVersion(gover); ver != nil {
			return ver, nil
		}
	}

	// Try to determine the version based on the schedinit function.
	if v := tryFromSchedInit(f); v != nil {
		return v, nil
	}

	// If no version was found, search the sections for the
	// version string.

	data, err := f.fh.getRData()
	// If a read-only data section does not exist, try text.
	if errors.Is(err, ErrSectionDoesNotExist) {
		_, data, err = f.fh.getCodeSection()
	}
	if err != nil {
		return nil, err
	}

	for {
		version := matchGoVersionString(data)
		if version == "" {
			return nil, ErrNoGoVersionFound
		}
		ver := ResolveGoVersion(version)
		// Go before 1.4 does not have the version string, so if we have found
		// a version string below 1.4beta1 it is a false positive.
		if ver == nil || GoVersionCompare(ver.Name, "go1.4beta1") < 0 {
			off := bytes.Index(data, []byte(version))
			// No match
			if off == -1 {
				break
			}
			data = data[off+2:]
			continue
		}
		return ver, nil
	}
	return nil, nil
}

func matchGoVersionString(data []byte) string {
	return string(goVersionMatcher.Find(data))
}
