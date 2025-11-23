//go:build wireinject
// +build wireinject

package main

import (
	"github.com/google/wire"
	"github.com/otelguard/otelguard/internal/config"
	internalwire "github.com/otelguard/otelguard/internal/wire"
)

// InitializeApplication creates a fully-wired Application instance.
// Wire will generate the implementation of this function.
func InitializeApplication(cfg *config.Config) (*internalwire.Application, error) {
	wire.Build(internalwire.ProviderSet)
	return nil, nil
}
