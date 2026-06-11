package tar

import (
	"fmt"
	"strings"
)

func parse(args []string) (options, error) {
	opts := options{destDir: "."}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "" {
			continue
		}
		if arg == "--overwrite" {
			opts.overwrite = true
			continue
		}
		if arg == "--skip-existing" {
			opts.overwrite = false
			continue
		}
		if arg == "-C" {
			i++
			opts.destDir = valueAt(args, i, "-C")
			continue
		}
		if len(arg) > 1 && arg[0] == '-' {
			if err := parseFlags(args, &i, strings.TrimPrefix(arg, "-"), &opts); err != nil {
				return opts, err
			}
			continue
		}
		opts.sourcePaths = append(opts.sourcePaths, arg)
	}
	if opts.mode == "" {
		return opts, fmt.Errorf("one of -c, -x, or -t is required")
	}
	if opts.archivePath == "" {
		return opts, fmt.Errorf("archive file is required")
	}
	if opts.mode == modeCreate && len(opts.sourcePaths) == 0 {
		return opts, fmt.Errorf("tar source path is required")
	}
	return opts, nil
}

func parseFlags(args []string, i *int, flags string, opts *options) error {
	for pos := 0; pos < len(flags); pos++ {
		switch flags[pos] {
		case 'c':
			if opts.mode != "" {
				return fmt.Errorf("only one tar mode is allowed")
			}
			opts.mode = modeCreate
		case 'x':
			if opts.mode != "" {
				return fmt.Errorf("only one tar mode is allowed")
			}
			opts.mode = modeExtract
		case 't':
			if opts.mode != "" {
				return fmt.Errorf("only one tar mode is allowed")
			}
			opts.mode = modeList
		case 'z':
			opts.gzip = true
		case 'f':
			if pos+1 < len(flags) {
				opts.archivePath = flags[pos+1:]
				return nil
			}
			*i = *i + 1
			opts.archivePath = valueAt(args, *i, "-f")
			return nil
		case 'C':
			if pos+1 < len(flags) {
				opts.destDir = flags[pos+1:]
				return nil
			}
			*i = *i + 1
			opts.destDir = valueAt(args, *i, "-C")
			return nil
		default:
			return fmt.Errorf("unsupported option: -%c", flags[pos])
		}
	}
	return nil
}

func valueAt(args []string, index int, name string) string {
	if index < 0 || index >= len(args) {
		panic(fmt.Sprintf("%s requires a value", name))
	}
	return args[index]
}
