//go:build !js && !wasm

package gore

import (
	"bytes"
	"golang.org/x/arch/x86/x86asm"
)

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

	if ok, err := f.fh.hasSymbolTable(); ok && err == nil {
		addr, size, err = f.fh.getSymbol("runtime.schedinit")
		if err == nil {
			goto disasm
		}
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
