// This file is part of GoRE.
//
// Copyright (C) 2019-2024 GoRE Authors
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

package main

import (
	"path/filepath"
	"text/template"
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

const moduleDataHeader = `
// This file is part of GoRE.
//
// Copyright (C) 2019-2023 GoRE Authors
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

package gore

`

var (
	goversionCsv         = filepath.Join(getSourceDir(), "resources", "goversions.csv")
	outputFile           = filepath.Join(getSourceDir(), "stdpkg_gen.go")
	goversionOutputFile  = filepath.Join(getSourceDir(), "goversion_gen.go")
	moduleDataOutputFile = filepath.Join(getSourceDir(), "moduledata_gen.go")
)
