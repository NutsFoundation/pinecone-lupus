package core

import (
	"context"
	"strings"

	"github.com/NutsFoundation/pinecone-lupus/etherium/mushroom/php/arch/entity"
)

func nonNilContext(ctx context.Context) context.Context {
	if ctx != nil {
		return ctx
	}
	return context.Background()
}

func normalizeConfig(config entity.MushroomConfig) entity.MushroomConfig {
	if strings.TrimSpace(config.Runtime.Host) == "" {
		config.Runtime.Host = "127.0.0.1"
	}
	if config.Runtime.Port <= 0 {
		config.Runtime.Port = 19070
	}
	if config.Runtime.StartupTimeoutMillis <= 0 {
		config.Runtime.StartupTimeoutMillis = 10000
	}
	if config.Runtime.RequestTimeoutMillis <= 0 {
		config.Runtime.RequestTimeoutMillis = 30000
	}
	if config.Runtime.MaxRequestBodyBytes <= 0 {
		config.Runtime.MaxRequestBodyBytes = 10 * 1024 * 1024
	}
	if config.Runtime.MaxResponseBodyBytes <= 0 {
		config.Runtime.MaxResponseBodyBytes = 10 * 1024 * 1024
	}
	if config.MaxBodyBytes <= 0 {
		config.MaxBodyBytes = config.Runtime.MaxRequestBodyBytes
	}
	return config
}
