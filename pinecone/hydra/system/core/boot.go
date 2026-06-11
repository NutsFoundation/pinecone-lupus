package core

import (
	"context"

	"github.com/NutsFoundation/pinecone-lupus/pinecone/hydra/system/arch"
)

func RunMorph(ctx context.Context, morph arch.Morph) error {
	if morph == nil {
		return nil
	}
	defer func() {
		_ = morph.Shutdown(ctx)
	}()
	return morph.Run(ctx)
}
