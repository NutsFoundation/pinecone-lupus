package arch

type Client interface {
	Objects() ObjectOperations
	Kernel() KernelOperations
}
