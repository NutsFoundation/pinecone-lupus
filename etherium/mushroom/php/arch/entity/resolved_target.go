package entity

type ResolvedTarget struct {
	DocumentRoot string
	ScriptPath   string
	ScriptName   string
	EntryPath    string
	ExtraParams  map[string]string
}
