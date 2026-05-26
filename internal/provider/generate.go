// Package provider hosts the generator hook for `go generate`. Documentation
// is produced by terraform-plugin-docs, which is pinned as a tool dependency
// in tools/tools.go. Running `go generate ./...` from the repository root
// regenerates the contents of the `docs/` directory from the provider schema
// and the templates/examples on disk.
package provider

//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate
