package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// The skeleton is shared with the Backstage Software Template, so rfctl understands the subset of
// Nunjucks used there: ${{ values.key }} and {%- if values.key ==|!= "v" %} ... {%- else %} ... {%- endif %}.
var (
	valueRe = regexp.MustCompile(`\$\{\{\s*values\.(\w+)\s*\}\}`)
	ifRe    = regexp.MustCompile(`^\s*\{%-?\s*if values\.(\w+)\s*(==|!=)\s*"([^"]*)"\s*%\}\s*$`)
	elseRe  = regexp.MustCompile(`^\s*\{%-?\s*else\s*%\}\s*$`)
	endRe   = regexp.MustCompile(`^\s*\{%-?\s*endif\s*%\}\s*$`)
)

type block struct{ active, parent bool }

// renderer evaluates one template: a stack of if-blocks and the output lines.
type renderer struct {
	values map[string]string
	stack  []block
	out    []string
}

// directive handles {% if %}, {% else %} and {% endif %} lines; it reports whether the line was a directive.
func (r *renderer) directive(line string) (bool, error) {
	top := r.stack[len(r.stack)-1]
	if m := ifRe.FindStringSubmatch(line); m != nil {
		cond := (r.values[m[1]] == m[3]) == (m[2] == "==")
		r.stack = append(r.stack, block{active: top.active && cond, parent: top.active})
		return true, nil
	}
	isElse, isEnd := elseRe.MatchString(line), endRe.MatchString(line)
	if !isElse && !isEnd {
		return false, nil
	}
	if len(r.stack) == 1 {
		return true, fmt.Errorf("%q without if", strings.TrimSpace(line))
	}
	if isElse {
		r.stack[len(r.stack)-1].active = top.parent && !top.active
	} else {
		r.stack = r.stack[:len(r.stack)-1]
	}
	return true, nil
}

// substitute replaces ${{ values.key }} placeholders.
func (r *renderer) substitute(line string) (string, error) {
	var missing string
	line = valueRe.ReplaceAllStringFunc(line, func(s string) string {
		key := valueRe.FindStringSubmatch(s)[1]
		v, ok := r.values[key]
		if !ok {
			missing = key
		}
		return v
	})
	if missing != "" {
		return "", fmt.Errorf("unknown value %q", missing)
	}
	return line, nil
}

// line processes one template line.
func (r *renderer) line(line string) error {
	isDirective, err := r.directive(line)
	if err != nil || isDirective || !r.stack[len(r.stack)-1].active {
		return err
	}
	line, err = r.substitute(line)
	if err == nil {
		r.out = append(r.out, line)
	}
	return err
}

// render applies values to a template text.
func render(text string, values map[string]string) (string, error) {
	r := &renderer{values: values, stack: []block{{active: true, parent: true}}}
	for i, line := range strings.Split(text, "\n") {
		if err := r.line(line); err != nil {
			return "", fmt.Errorf("line %d: %w", i+1, err)
		}
	}
	if len(r.stack) != 1 {
		return "", fmt.Errorf("unterminated if block")
	}
	return strings.Join(r.out, "\n"), nil
}

// renderFile renders one template file of src into dst and returns its relative path.
func renderFile(src, dst, path string, values map[string]string) (string, error) {
	rel, _ := filepath.Rel(src, path)
	raw, err := os.ReadFile(path) //nolint:gosec // template files of the platform repository
	if err != nil {
		return "", err
	}
	text, err := render(string(raw), values)
	if err != nil {
		return "", fmt.Errorf("%s: %w", rel, err)
	}
	target := filepath.Join(dst, rel)
	if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
		return "", err
	}
	return rel, os.WriteFile(target, []byte(text), 0o600)
}

// renderDir renders every file of src into dst.
func renderDir(src, dst string, values map[string]string) ([]string, error) {
	var written []string
	err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := renderFile(src, dst, path, values)
		written = append(written, rel)
		return err
	})
	return written, err
}
