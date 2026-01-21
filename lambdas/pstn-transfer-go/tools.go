//go:build tools
// +build tools

// This file follows
// https://go.dev/wiki/Modules#how-can-i-track-tool-dependencies-for-a-module

package tools

import (
	_ "github.com/blmayer/awslambdarpc"
)
