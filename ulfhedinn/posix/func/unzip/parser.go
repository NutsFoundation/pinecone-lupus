package unzip

import "fmt"

func parse(args []string) (options, error) {
	opts := options{destDir: "."}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "" {
			continue
		}
		if arg == "-d" {
			i++
			opts.destDir = valueAt(args, i, "-d")
			continue
		}
		if len(arg) > 1 && arg[0] == '-' {
			flags := arg[1:]
			for pos := 0; pos < len(flags); pos++ {
				switch flags[pos] {
				case 'q':
					opts.quiet = true
				case 'o':
					opts.overwrite = true
				case 'l':
					opts.listOnly = true
				case 'd':
					if pos+1 < len(flags) {
						opts.destDir = flags[pos+1:]
						pos = len(flags)
					} else {
						i++
						opts.destDir = valueAt(args, i, "-d")
					}
				default:
					return opts, fmt.Errorf("unsupported option: -%c", flags[pos])
				}
			}
			continue
		}
		if opts.sourcePath == "" {
			opts.sourcePath = arg
			continue
		}
		return opts, fmt.Errorf("unexpected argument: %s", arg)
	}
	if opts.sourcePath == "" {
		return opts, fmt.Errorf("zip file is required")
	}
	return opts, nil
}

func valueAt(args []string, index int, name string) string {
	if index < 0 || index >= len(args) {
		panic(fmt.Sprintf("%s requires a value", name))
	}
	return args[index]
}
