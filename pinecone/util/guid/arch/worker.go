package arch

type WorkerIdAssigner interface {
	AssignWorkerId() int64
}
