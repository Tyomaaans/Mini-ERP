package docs

import _ "embed"

//go:embed openapi.yaml
var Spec []byte

//go:embed index.html
var UI []byte
