package aic

import (
	"fmt"
	"strconv"
	"strings"
)

func parseIntPairArgs(args string) (int, int, error) {
	parts := strings.Split(args, ",")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("expected two integers")
	}
	x, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return 0, 0, err
	}
	y, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return 0, 0, err
	}
	return x, y, nil
}

func parseOneArg(args string) (string, error) {
	args = strings.TrimSpace(args)
	if args == "" {
		return "", fmt.Errorf("missing argument")
	}
	if strings.HasPrefix(args, `"`) {
		list, err := parseMultiStringArgs(args)
		if err != nil {
			return "", err
		}
		if len(list) != 1 {
			return "", fmt.Errorf("expected exactly one argument")
		}
		return list[0], nil
	}
	if strings.Contains(args, ",") {
		return "", fmt.Errorf("expected single argument")
	}
	return args, nil
}

func parseMultiStringArgs(args string) ([]string, error) {
	var results []string
	rest := strings.TrimSpace(args)
	if rest == "" {
		return []string{}, nil
	}

	for len(rest) > 0 {
		if rest[0] != '"' {
			return nil, fmt.Errorf("expected '\"' at start of argument, got %q", rest[0])
		}

		var buf strings.Builder
		escaped := false
		i := 1
		foundEnd := false

		for i < len(rest) {
			ch := rest[i]

			if escaped {
				switch ch {
				case '"':
					buf.WriteByte('"')
				case '\\':
					buf.WriteByte('\\')
				case 'n':
					buf.WriteByte('\n')
				case 't':
					buf.WriteByte('\t')
				case 'r':
					buf.WriteByte('\r')
				default:
					buf.WriteByte(ch)
				}
				escaped = false
				i++
				continue
			}

			if ch == '\\' {
				escaped = true
				i++
				continue
			}

			if ch == '"' {
				foundEnd = true
				i++
				break
			}

			buf.WriteByte(ch)
			i++
		}

		if !foundEnd {
			return nil, fmt.Errorf("unterminated string")
		}

		results = append(results, buf.String())
		rest = strings.TrimSpace(rest[i:])

		if len(rest) == 0 {
			break
		}
		if rest[0] != ',' {
			return nil, fmt.Errorf("expected comma between arguments")
		}
		rest = strings.TrimSpace(rest[1:])
	}

	return results, nil
}
