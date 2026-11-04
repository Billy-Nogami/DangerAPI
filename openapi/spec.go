// Package openapi предоставляет OpenAPI-спецификацию сервиса.
package openapi

import _ "embed"

// Specification содержит исходную OpenAPI-спецификацию, встроенную в исполняемый файл.
//
//go:embed openapi.yaml
var Specification []byte
