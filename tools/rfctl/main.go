// Command rfctl is the RoomFlow Internal Developer Platform CLI (lab 11): it creates services from the
// golden-path template, requests infrastructure (Crossplane claims), validates catalog descriptors and
// shows deployment status. Every command supports -o json for scripting.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

const usage = `rfctl — CLI платформи RoomFlow

Команди:
  login     --token T                         зберегти токен GitHub (~/.config/rfctl/token)
  new       --name N --lang go|python --db none|postgres|redis --owner O [--out DIR]
  provision --kind postgres --name N [--size small|medium] [--namespace NS]
  catalog   validate [FILE]
  status    [--repo owner/name]                стан останніх запусків CI/CD
Загальні прапорці: -o text|json
`

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		_, _ = fmt.Fprint(stderr, usage)
		return 2
	}
	cmds := map[string]func([]string) (any, error){
		"login": cmdLogin, "new": cmdNew, "provision": cmdProvision, "catalog": cmdCatalog, "status": cmdStatus,
	}
	cmd, ok := cmds[args[0]]
	if !ok {
		_, _ = fmt.Fprint(stderr, usage)
		return 2
	}
	rest, format := outputFormat(args[1:])
	res, err := cmd(rest)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	if format == "json" {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(res)
		return 0
	}
	if s, ok := res.(fmt.Stringer); ok {
		_, _ = fmt.Fprintln(stdout, s.String())
	} else {
		_, _ = fmt.Fprintf(stdout, "%v\n", res)
	}
	return 0
}

// outputFormat extracts -o/--output from args.
func outputFormat(args []string) ([]string, string) {
	var rest []string
	format := "text"
	for i := 0; i < len(args); i++ {
		if (args[i] == "-o" || args[i] == "--output") && i+1 < len(args) {
			format = args[i+1]
			i++
			continue
		}
		rest = append(rest, args[i])
	}
	return rest, format
}
