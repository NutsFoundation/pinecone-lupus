package core

import (
	"context"

	"github.com/NutsFoundation/pinecone-lupus/etherium/mushroom/php/arch"
	"github.com/NutsFoundation/pinecone-lupus/etherium/mushroom/php/arch/entity"
)

type Gateway struct {
	runtime arch.RuntimeManager
	fastcgi *FastCGIClient
}

var _ arch.Gateway = (*Gateway)(nil)

func NewGateway(config entity.MushroomConfig, runtime arch.RuntimeManager) *Gateway {
	return &Gateway{
		runtime: runtime,
		fastcgi: NewFastCGIClient(config),
	}
}

func (g *Gateway) Serve(ctx context.Context, request *entity.MushroomRequest, target *entity.ResolvedTarget) (*entity.MushroomResponse, error) {
	if g.runtime != nil {
		if err := g.runtime.EnsureReady(ctx); err != nil {
			return nil, err
		}
	}
	return g.fastcgi.Execute(ctx, request, target)
}
