package cli

import "io"

type Invocation struct {
	Args    []string
	Env     map[string]string
	WorkDir string
	Stdout  io.Writer
	Stderr  io.Writer
}
