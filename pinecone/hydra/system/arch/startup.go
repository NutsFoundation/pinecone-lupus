package arch

import "io"

type StartupMode string

type StartupCommand struct {
	Mode       StartupMode
	ConfigPath string
	RawArgs    []string
}

type MorphContext struct {
	Mode       StartupMode
	ConfigPath string
	RawArgs    []string
	Stdin      io.Reader
	Stdout     io.Writer
	Stderr     io.Writer
}
