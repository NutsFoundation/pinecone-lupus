package zip

import "fmt"

func parse(args []string) (options, error) {
	opts := options{}
	for _, arg := range args {
		if arg == "" {
			continue
		}
		if len(arg) > 1 && arg[0] == '-' {
			flags := arg[1:]
			for pos := 0; pos < len(flags); pos++ {
				switch flags[pos] {
				case 'r':
					opts.recursive = true
				case 'q':
					opts.quiet = true
				default:
					return opts, fmt.Errorf("unsupported option: -%c", flags[pos])
				}
			}
			continue
		}
		if opts.outputPath == "" {
			opts.outputPath = arg
		} else {
			opts.sourcePaths = append(opts.sourcePaths, arg)
		}
	}
	if opts.outputPath == "" {
		return opts, fmt.Errorf("zip output file is required")
	}
	if len(opts.sourcePaths) == 0 {
		return opts, fmt.Errorf("zip source path is required")
	}
	return opts, nil
}
