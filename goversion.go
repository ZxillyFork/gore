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

	"golang.org/x/arch/x86/x86asm"

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

// tryFromSchedInit tries to identify the version of the Go compiler that compiled the code.
// The function "schedinit" in the "runtime" package has the only reference to this string
// used to identify the version.
// The function returns nil if no version is found.
func tryFromSchedInit(f *GoFile) *GoVersion {
	// Check for non-supported architectures.
	if f.FileInfo.Arch != Arch386 && f.FileInfo.Arch != ArchAMD64 {
		return nil
	}

	var addr, size uint64
	var fcn *Function
	var std []*Package
	var err error

	is32 := false
	if f.FileInfo.Arch == Arch386 {
		is32 = true
	}

	sym, err := f.fh.getSymbol("runtime.schedinit")
	if err == nil {
		addr = sym.Value
		size = sym.Size
		goto disasm
	}

	// Find schedinit function.
	std, err = f.GetSTDLib()
	if err != nil {
		return nil
	}

pkgLoop:
	for _, v := range std {
		if v.Name != "runtime" {
			continue
		}
		for _, vv := range v.Functions {
			if vv.Name != "schedinit" {
				continue
			}
			fcn = vv
			break pkgLoop
		}
	}

	// Check if the function was found
	if fcn == nil {
		// If we can't find the function, there is nothing to do.
		return nil
	}
	addr = fcn.Offset
	size = fcn.End - fcn.Offset

disasm:
	// Get the raw hex.
	buf, err := f.Bytes(addr, size)
	if err != nil {
		return nil
	}

	/*
		Disassemble the function until the loading of the Go version is found.
	*/

	// Counter for how many bytes has been read.
	s := 0
	mode := f.FileInfo.WordSize * 8

	for s < len(buf) {
		inst, err := x86asm.Decode(buf[s:], mode)
		if err != nil {
			// If we fail to decode the instruction, something is wrong so
			// bailout.
			return nil
		}

		// Update next instruction location.
		s = s + inst.Len

		// Check if it's a "lea" instruction.
		if inst.Op != x86asm.LEA {
			continue
		}

		// Check what it's loading and if it's pointing to the compiler version used.
		// First assume that the address is a direct addressing.
		arg := inst.Args[1].(x86asm.Mem)
		disp := arg.Disp
		if arg.Base == x86asm.EIP || arg.Base == x86asm.RIP {
			// If the addressing is based on the instruction pointer, fix the address.
			disp += int64(addr) + int64(s)
		}

		// If the addressing is based on the stack pointer, this is not the right
		// instruction.
		if arg.Base == x86asm.ESP || arg.Base == x86asm.RSP {
			continue
		}

		// Resolve the pointer to the string. If we get no data, this is not the
		// right instruction.
		b, _ := f.Bytes(uint64(disp), uint64(0x20))
		if b == nil {
			continue
		}

		r := bytes.NewReader(b)
		ptr, err := readUIntTo64(r, f.FileInfo.ByteOrder, is32)
		if err != nil {
			// Probably not the right instruction, so go to next.
			continue
		}
		l, err := readUIntTo64(r, f.FileInfo.ByteOrder, is32)
		if err != nil {
			// Probably not the right instruction, so go to next.
			continue
		}

		bstr, _ := f.Bytes(ptr, l)
		if bstr == nil {
			continue
		}

		if !bytes.HasPrefix(bstr, []byte("go1.")) {
			continue
		}

		// Likely the version string.
		ver := string(bstr)

		resolvedVer := ResolveGoVersion(ver)
		if resolvedVer != nil {
			return resolvedVer
		}

		// An unknown version.
		return &GoVersion{Name: ver}
	}

	return nil
}

func matchGoVersionString(data []byte) string {
	return string(goVersionMatcher.Find(data))
}
