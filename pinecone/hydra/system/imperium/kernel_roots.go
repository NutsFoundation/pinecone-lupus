package imperium

var KernelRoots = map[string]struct{}{
	ConfRoot: {},
	DevRoot:  {},
	HomeRoot: {},
	MntRoot:  {},
	SysRoot:  {},
	ProcRoot: {},
	VarRoot:  {},
	MetaRoot: {},
}

func IsKernelRoot(root string) bool {
	_, ok := KernelRoots[root]
	return ok
}
