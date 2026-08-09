// This module boundary exists solely to keep the root Go module's ./...
// pattern from descending into web/node_modules, where some npm packages
// (e.g. flatted) ship stray vendored .go files that are irrelevant to
// maelsink and would otherwise get picked up by go build/vet/test.
module github.com/0funct0ry/maelsink/web

go 1.26.4
