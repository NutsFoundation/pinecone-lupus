package wget

import (
	"fmt"
	"path"
	"strconv"
	"strings"
)

func parse(args []string) (options, error) {
	opts := options{headers: map[string]string{}}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "" {
			continue
		}
		if strings.HasPrefix(arg, "--") {
			key, value, hasValue := strings.Cut(arg, "=")
			switch key {
			case "--header":
				if !hasValue {
					i++
					value = valueAt(args, i, key)
				}
				if err := addHeader(opts.headers, value); err != nil {
					return opts, err
				}
			case "--timeout":
				if !hasValue {
					i++
					value = valueAt(args, i, key)
				}
				opts.timeoutMillis = secondsToMillis(value, key)
			default:
				return opts, fmt.Errorf("unsupported option: %s", key)
			}
			continue
		}
		if strings.HasPrefix(arg, "-") && arg != "-" {
			if err := parseShort(args, &i, arg, &opts); err != nil {
				return opts, err
			}
			continue
		}
		opts.url = arg
	}
	if opts.url == "" {
		return opts, fmt.Errorf("URL is required")
	}
	if opts.outputPath == "" {
		opts.outputPath = remoteName(opts.url)
	}
	return opts, nil
}

func parseShort(args []string, i *int, arg string, opts *options) error {
	flags := arg[1:]
	for pos := 0; pos < len(flags); pos++ {
		switch flags[pos] {
		case 'q':
			opts.quiet = true
		case 'O':
			if pos+1 < len(flags) {
				opts.outputPath = flags[pos+1:]
				return nil
			}
			*i = *i + 1
			opts.outputPath = valueAt(args, *i, "-O")
			return nil
		default:
			return fmt.Errorf("unsupported option: -%c", flags[pos])
		}
	}
	return nil
}

func addHeader(headers map[string]string, header string) error {
	key, value, ok := strings.Cut(header, ":")
	if !ok || strings.TrimSpace(key) == "" {
		return fmt.Errorf("invalid header: %s", header)
	}
	headers[strings.TrimSpace(key)] = strings.TrimSpace(value)
	return nil
}

func secondsToMillis(value string, name string) int64 {
	seconds, err := strconv.ParseFloat(value, 64)
	if err != nil || seconds < 0 {
		panic(fmt.Sprintf("invalid %s: %s", name, value))
	}
	return int64(seconds * 1000)
}

func valueAt(args []string, index int, name string) string {
	if index < 0 || index >= len(args) {
		panic(fmt.Sprintf("%s requires a value", name))
	}
	return args[index]
}

func remoteName(rawURL string) string {
	clean := strings.TrimRight(rawURL, "/")
	name := path.Base(clean)
	if name == "." || name == "/" || name == "" {
		return "index.html"
	}
	return name
}
