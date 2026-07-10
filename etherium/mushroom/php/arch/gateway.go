package arch

import (
	"context"

	"github.com/NutsFoundation/pinecone-lupus/etherium/mushroom/php/arch/entity"
)

type Gateway interface {
	Serve(ctx context.Context, request *entity.MushroomRequest, target *entity.ResolvedTarget) (*entity.MushroomResponse, error)
}
