//go:build tools

// Package tools pins build-time dependencies that are not directly imported by
// the provider but are required for documentation generation and release
// workflows. See https://github.com/golang/go/wiki/Modules#how-can-i-track-tool-dependencies-for-a-module
package tools

import (
	_ "github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs"
)
