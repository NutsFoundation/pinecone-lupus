package core

import (
	"os"

	"github.com/NutsFoundation/pinecone-lupus/pinecone/hydra/system/arch"
)

func EnsureRuntimeDirs(runtime arch.Runtime) error {
	if runtime == nil {
		return nil
	}
	for _, dir := range []string{
		runtime.RootPath(),
		runtime.SetupPath(),
		runtime.LogsPath(),
		runtime.TempPath(),
	} {
		if dir == "" {
			continue
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return nil
}
