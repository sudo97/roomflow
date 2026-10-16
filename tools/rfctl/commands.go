package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Template value and catalog field names.
const (
	keyName  = "name"
	keyOwner = "owner"
	keyDesc  = "description"
	keyType  = "type"
	keyLife  = "lifecycle"
)

var nameRe = regexp.MustCompile(`^[a-z][a-z0-9-]{2,30}$`)

func tokenPath() string {
	dir, _ := os.UserConfigDir()
	return filepath.Join(dir, "rfctl", "token")
}

func token() string {
	if t := os.Getenv("GITHUB_TOKEN"); t != "" {
		return t
	}
	b, _ := os.ReadFile(tokenPath())
	return strings.TrimSpace(string(b))
}

type message struct {
	Status  string   `json:"status"`
	Message string   `json:"message"`
	Files   []string `json:"files,omitempty"`
}

func (m message) String() string {
	s := m.Message
	for _, f := range m.Files {
		s += "\n  " + f
	}
	return s
}

func cmdLogin(args []string) (any, error) {
	fs := flag.NewFlagSet("login", flag.ContinueOnError)
	tok := fs.String("token", "", "GitHub token")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if *tok == "" {
		return nil, errors.New("--token is required")
	}
	if err := os.MkdirAll(filepath.Dir(tokenPath()), 0o700); err != nil {
		return nil, err
	}
	if err := os.WriteFile(tokenPath(), []byte(*tok+"\n"), 0o600); err != nil {
		return nil, err
	}
	return message{Status: "ok", Message: "token saved to " + tokenPath()}, nil
}

func cmdNew(args []string) (any, error) {
	fs := flag.NewFlagSet("new", flag.ContinueOnError)
	name := fs.String(keyName, "", "service name")
	lang := fs.String("lang", "go", "go|python")
	db := fs.String("db", "none", "none|postgres|redis")
	owner := fs.String(keyOwner, "group:platform-team", "owner")
	desc := fs.String(keyDesc, "Сервіс платформи RoomFlow", "description")
	out := fs.String("out", "", "output directory (default ./NAME)")
	tmpl := fs.String("template", "platform/templates/service/skeleton", "skeleton directory")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if err := validateNew(*name, *lang, *db); err != nil {
		return nil, err
	}
	dst := *out
	if dst == "" {
		dst = *name
	}
	values := map[string]string{keyName: *name, "language": *lang, "database": *db, keyOwner: *owner, keyDesc: *desc}
	var files []string
	for _, part := range []string{"common", *lang} {
		f, err := renderDir(filepath.Join(*tmpl, part), dst, values)
		if err != nil {
			return nil, err
		}
		files = append(files, f...)
	}
	return message{Status: "created", Message: fmt.Sprintf("service %s created in %s", *name, dst), Files: files}, nil
}

var (
	languages = map[string]bool{"go": true, "python": true}
	databases = map[string]bool{"none": true, "postgres": true, "redis": true}
)

func validateNew(name, lang, db string) error {
	switch {
	case !nameRe.MatchString(name):
		return fmt.Errorf("invalid name %q: expected %s", name, nameRe)
	case !languages[lang]:
		return fmt.Errorf("unsupported language %q", lang)
	case !databases[db]:
		return fmt.Errorf("unsupported database %q", db)
	}
	return nil
}

const claimTemplate = `apiVersion: platform.roomflow.io/v1alpha1
kind: PostgreSQLInstance
metadata:
  name: %s
  namespace: %s
spec:
  parameters: { size: %s, storageGB: %d }
  compositionSelector: { matchLabels: { provider: kubernetes } }
`

type claim struct {
	Kind     string `json:"kind"`
	Manifest string `json:"manifest"`
}

func (c claim) String() string { return c.Manifest }

func cmdProvision(args []string) (any, error) {
	fs := flag.NewFlagSet("provision", flag.ContinueOnError)
	kind := fs.String("kind", "postgres", "resource kind")
	name := fs.String(keyName, "", "resource name")
	size := fs.String("size", "small", "small|medium")
	ns := fs.String("namespace", "default", "namespace")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if *kind != "postgres" {
		return nil, fmt.Errorf("unsupported kind %q (available: postgres)", *kind)
	}
	if !nameRe.MatchString(*name) {
		return nil, fmt.Errorf("invalid name %q", *name)
	}
	storage := map[string]int{"small": 1, "medium": 5}[*size]
	if storage == 0 {
		return nil, fmt.Errorf("unsupported size %q", *size)
	}
	return claim{Kind: "PostgreSQLInstance", Manifest: fmt.Sprintf(claimTemplate, *name, *ns, *size, storage)}, nil
}

type validation struct {
	File     string   `json:"file"`
	Entities int      `json:"entities"`
	Problems []string `json:"problems"`
}

func (v validation) String() string {
	if len(v.Problems) == 0 {
		return fmt.Sprintf("%s: %d entities, OK", v.File, v.Entities)
	}
	return fmt.Sprintf("%s: %d entities, problems:\n  %s", v.File, v.Entities, strings.Join(v.Problems, "\n  "))
}

// cmdCatalog validates the minimal golden-path contract of catalog descriptors.
func cmdCatalog(args []string) (any, error) {
	if len(args) == 0 || args[0] != "validate" {
		return nil, errors.New("usage: rfctl catalog validate [FILE]")
	}
	file := "catalog-info.yaml"
	if len(args) > 1 {
		file = args[1]
	}
	raw, err := os.ReadFile(file) //nolint:gosec // file chosen by the CLI user
	if err != nil {
		return nil, err
	}
	res := validation{File: file, Problems: []string{}}
	for i, doc := range strings.Split(string(raw), "\n---") {
		if strings.TrimSpace(doc) != "" {
			res.Entities++
			res.Problems = append(res.Problems, problems(i+1, doc)...)
		}
	}
	if len(res.Problems) > 0 {
		return res, fmt.Errorf("%s", res.String())
	}
	return res, nil
}

// problems lists missing required fields of one entity.
func problems(n int, doc string) []string {
	var out []string
	kind := field(doc, "kind")
	for _, req := range requiredFields(kind) {
		if field(doc, req) == "" {
			out = append(out, fmt.Sprintf("entity %d (%s %s): missing %s", n, kind, field(doc, keyName), req))
		}
	}
	return out
}

func requiredFields(kind string) []string {
	switch kind {
	case "Component":
		return []string{keyName, keyDesc, keyType, keyLife, keyOwner, "system"}
	case "API":
		return []string{keyName, keyType, keyLife, keyOwner, "definition"}
	case "Resource", "System":
		return []string{keyName, keyOwner}
	default:
		return []string{keyName}
	}
}

// field returns the value of the first "key:" line in a YAML document (sufficient for catalog files).
func field(doc, key string) string {
	for _, line := range strings.Split(doc, "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, key+":") {
			v := strings.TrimSpace(strings.TrimPrefix(t, key+":"))
			if v == "" {
				return "(block)"
			}
			return v
		}
	}
	return ""
}

type runStatus struct {
	Workflow   string    `json:"workflow"`
	Status     string    `json:"status"`
	Conclusion string    `json:"conclusion"`
	Commit     string    `json:"commit"`
	Updated    time.Time `json:"updated"`
}

type statusReport []runStatus

func (s statusReport) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%-28s %-10s %-10s %-8s %s\n", "WORKFLOW", "STATUS", "RESULT", "COMMIT", "UPDATED")
	for _, r := range s {
		fmt.Fprintf(&b, "%-28s %-10s %-10s %-8s %s\n", r.Workflow, r.Status, r.Conclusion, r.Commit, r.Updated.Format(time.RFC3339))
	}
	return strings.TrimRight(b.String(), "\n")
}

var githubAPI = "https://api.github.com"

type ghRun struct {
	Name       string    `json:"name"`
	Status     string    `json:"status"`
	Conclusion string    `json:"conclusion"`
	HeadSHA    string    `json:"head_sha"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func fetchRuns(repo string) ([]ghRun, error) {
	req, err := http.NewRequest(http.MethodGet, githubAPI+"/repos/"+repo+"/actions/runs?branch=main&per_page=50", nil)
	if err != nil {
		return nil, err
	}
	if t := token(); t != "" {
		req.Header.Set("Authorization", "Bearer "+t)
	}
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github: status %d", resp.StatusCode)
	}
	var body struct {
		Runs []ghRun `json:"workflow_runs"`
	}
	err = json.NewDecoder(resp.Body).Decode(&body)
	return body.Runs, err
}

// cmdStatus shows the latest run of every workflow on main (deployment status of the service).
func cmdStatus(args []string) (any, error) {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	repo := fs.String("repo", "sudo97/roomflow", "owner/name")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	runs, err := fetchRuns(*repo)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var out statusReport
	for _, r := range runs {
		if !seen[r.Name] {
			seen[r.Name] = true
			out = append(out, runStatus{Workflow: r.Name, Status: r.Status, Conclusion: r.Conclusion, Commit: r.HeadSHA[:min(7, len(r.HeadSHA))], Updated: r.UpdatedAt})
		}
	}
	return out, nil
}
