package core

import "github.com/NutsFoundation/pinecone-lupus/pinecone/hydra/system/arch"

type BasicRuntime struct {
	RootAbsPath  string
	SetupAbsPath string
	LogsAbsPath  string
	TempAbsPath  string
}

var _ arch.Runtime = (*BasicRuntime)(nil)

func NewBasicRuntime(rootAbsPath, setupAbsPath, logsAbsPath, tempAbsPath string) *BasicRuntime {
	return &BasicRuntime{
		RootAbsPath:  rootAbsPath,
		SetupAbsPath: setupAbsPath,
		LogsAbsPath:  logsAbsPath,
		TempAbsPath:  tempAbsPath,
	}
}

func (r *BasicRuntime) RootPath() string {
	if r == nil {
		return ""
	}
	return r.RootAbsPath
}

func (r *BasicRuntime) SetupPath() string {
	if r == nil {
		return ""
	}
	return r.SetupAbsPath
}

func (r *BasicRuntime) LogsPath() string {
	if r == nil {
		return ""
	}
	return r.LogsAbsPath
}

func (r *BasicRuntime) TempPath() string {
	if r == nil {
		return ""
	}
	return r.TempAbsPath
}
