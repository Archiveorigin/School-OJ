package openapi

import _ "embed"

// document is the HTTP contract used by both the API route audit and the Web client generator.
//
//go:embed openapi.json
var document []byte

func Document() []byte {
	return document
}
