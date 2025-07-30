//go:build tools
// +build tools

package tools

import (
	_ "github.com/golang/mock/mockgen/model"
	_ "github.com/golangci/golangci-lint/cmd/golangci-lint"
	_ "github.com/goreleaser/goreleaser"
	_ "github.com/katallaxie/pkg/cmd/runproc"
	_ "gotest.tools/gotestsum"
	_ "k8s.io/code-generator"
	_ "mvdan.cc/gofumpt"
	_ "sigs.k8s.io/kustomize/kustomize/v5"
)
