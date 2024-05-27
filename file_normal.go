//go:build !js && !wasm

package gore

// GetGoRoot returns the Go Root path used to compile the binary.
func (f *GoFile) GetGoRoot() (string, error) {
	err := f.initPackages()
	if err != nil {
		return "", err
	}
	return findGoRootPath(f)
}
