package defaultcontent

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"unicode"

	"github.com/luckylhb90/bnbloc/core"
)

// The content kinds (bnbloc's content package format).
const (
	kindExpert = "bnbloc.expert.package/v1"
	kindSkill  = "bnbloc.skill.package/v1"
	kindPrompt = "bnbloc.prompt.package/v1"
	kindAssist = "bnbloc.assist.package/v1"
)

// field is an assist's reply field (component.json fields[]).
type field struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Type        string  `json:"type"`
	Item        []field `json:"item"`
}

// prompt is a prompt pack's prompt.
type prompt struct {
	ID        string                         `json:"id"`
	Title     string                         `json:"title"`
	Body      string                         `json:"body"`
	Slash     string                         `json:"slash"`
	Starter   bool                           `json:"starter"`
	Files     *struct{ Extensions []string } `json:"files"`
	Selection []string                       `json:"selection"`
	Variables []json.RawMessage              `json:"variables"`
}

// manifest is what the tests read of a component.json (the platform's
// validator reads all of it: bnbloc content check).
type manifest struct {
	Kind     string `json:"kind"`
	Name     string `json:"name"`
	Version  string `json:"version"`
	Requires struct {
		Skills  []string `json:"skills"`
		Prompts []string `json:"prompts"`
		MCP     []struct {
			Name string `json:"name"`
		} `json:"mcp"`
	} `json:"requires"`
	QuickPrompts []struct {
		Title  string `json:"title"`
		Prompt string `json:"prompt"`
		Ref    string `json:"ref"`
	} `json:"quickPrompts"`
	Prompts   []prompt `json:"prompts"`
	EveryRoom bool     `json:"everyRoom"`
	Output    string   `json:"output"`
	Context   []string `json:"context"`
	Fields    []field  `json:"fields"`
	Gate      *struct {
		Question  string  `json:"question"`
		Threshold float64 `json:"threshold"`
	} `json:"gate"`
}

// pkg is one package directory of packages/.
type pkg struct {
	Dir      string
	Manifest manifest
	Files    map[string]string
}

func (p pkg) file(t *testing.T, name string) string {
	t.Helper()
	s, ok := p.Files[name]
	if !ok {
		t.Fatalf("%s has no %s", p.Manifest.Name, name)
	}
	return s
}

var (
	loadOnce sync.Once
	loaded   map[string]pkg
	loadErr  error
)

// packages reads every package of packages/ by name (hidden files aside,
// as the platform's packer leaves them out).
func packages(t *testing.T) map[string]pkg {
	t.Helper()
	loadOnce.Do(func() {
		loaded = map[string]pkg{}
		entries, err := os.ReadDir("packages")
		if err != nil {
			loadErr = err
			return
		}
		for _, e := range entries {
			if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
				continue
			}
			dir := filepath.Join("packages", e.Name())
			p := pkg{Dir: dir, Files: map[string]string{}}
			loadErr = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
				if err != nil || d.IsDir() || strings.HasPrefix(d.Name(), ".") {
					return err
				}
				raw, err := os.ReadFile(path)
				rel, _ := filepath.Rel(dir, path)
				p.Files[filepath.ToSlash(rel)] = string(raw)
				return err
			})
			if loadErr != nil {
				return
			}
			if loadErr = json.Unmarshal([]byte(p.Files["component.json"]), &p.Manifest); loadErr != nil {
				return
			}
			loaded[e.Name()] = p
		}
	})
	if loadErr != nil {
		t.Fatal(loadErr)
	}
	return loaded
}

// get is the package named name.
func get(t *testing.T, name string) pkg {
	t.Helper()
	p, ok := packages(t)[name]
	if !ok {
		t.Fatalf("the %s package is missing", name)
	}
	return p
}

// bnbloc runs the pinned platform's CLI (go.mod's tool).
func bnbloc(t *testing.T, args ...string) []byte {
	t.Helper()
	cmd := exec.Command("go", append([]string{"tool", "bnbloc"}, args...)...)
	cmd.Env = append(os.Environ(), "GOWORK=off")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil && len(out) == 0 {
		t.Fatalf("bnbloc %s: %v\n%s", strings.Join(args, " "), err, stderr.String())
	}
	return out
}

// checkResult is one directory of `bnbloc content check --json`.
type checkResult struct {
	Dir     string `json:"dir"`
	Kind    string `json:"kind"`
	Name    string `json:"name"`
	Version string `json:"version"`
	Error   string `json:"error"`
}

// check validates directories with the platform's validator, as a save
// and a platform install would.
func check(t *testing.T, dirs ...string) []checkResult {
	t.Helper()
	var out []checkResult
	if err := json.Unmarshal(bnbloc(t, append([]string{"content", "check", "--json"}, dirs...)...), &out); err != nil {
		t.Fatal(err)
	}
	return out
}

// Every package validates with the pinned platform's validator, as its
// platform install will (an assist under a task's name meets the task's
// output contract), and its directory is its name.
func TestPackagesValidate(t *testing.T) {
	var dirs []string
	for _, p := range packages(t) {
		dirs = append(dirs, p.Dir)
	}
	sort.Strings(dirs)
	results := check(t, dirs...)
	if len(results) != len(dirs) {
		t.Fatalf("%d results for %d packages", len(results), len(dirs))
	}
	for _, r := range results {
		if r.Error != "" {
			t.Errorf("%s does not validate: %s", r.Dir, r.Error)
			continue
		}
		if filepath.Base(r.Dir) != r.Name {
			t.Errorf("%s names %q: the directory is the package's name", r.Dir, r.Name)
		}
	}
}

// No package names a workspace path of its own: what bnbloc writes for the
// agent and what a harness executes are the platform's lists
// (core.ProtectedWorkspacePaths); content refers to "bnbloc's room
// records" and the builder's drafts are never in them.
func TestNoWorkspacePaths(t *testing.T) {
	for name, p := range packages(t) {
		for f, text := range p.Files {
			for _, path := range core.ProtectedWorkspacePaths() {
				if strings.Contains(text, path) {
					t.Errorf("%s/%s names the workspace path %q: refer to bnbloc's and the harness's files generically", name, f, path)
				}
			}
		}
	}
	for _, rel := range []string{"drafts/contract-reviewer/component.json", "drafts/contract-reviewer/INSTRUCTIONS.md", "drafts/contract-reviewer/skills/clause-check/SKILL.md"} {
		if core.ProtectedWorkspacePath(rel) {
			t.Errorf("the draft path %s is protected: it would never be packed", rel)
		}
	}
}

// The default content hard-codes no human language besides its neutral
// English: the agent answers in the person's language, and an
// organization authors localized copies.
func TestNoLanguageHardCoded(t *testing.T) {
	for name, p := range packages(t) {
		for f, text := range p.Files {
			for _, r := range text {
				if unicode.In(r, unicode.Han, unicode.Hiragana, unicode.Katakana, unicode.Hangul, unicode.Cyrillic, unicode.Arabic) {
					t.Errorf("%s/%s hard-codes %q", name, f, r)
					break
				}
			}
		}
	}
}

// platformDir is the pinned platform's source tree (go.mod's version).
func platformDir(t *testing.T) string {
	t.Helper()
	cmd := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", "github.com/luckylhb90/bnbloc")
	cmd.Env = append(os.Environ(), "GOWORK=off")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list the platform: %v", err)
	}
	return strings.TrimSpace(string(out))
}

// The prompt texts live only in their packages: the platform's web reads
// the builder's quick prompts and the starters from the content API,
// never from a copy that drifts. A copy is a needle of a prompt's first
// characters in the pinned platform's web sources.
func TestPromptsStayOutOfThePlatform(t *testing.T) {
	const needleLen = 40
	var needles []string
	for _, p := range packages(t) {
		var texts []string
		for _, q := range p.Manifest.QuickPrompts {
			texts = append(texts, q.Prompt)
		}
		for _, pr := range p.Manifest.Prompts {
			texts = append(texts, pr.Body)
		}
		for _, s := range texts {
			s = strings.TrimSpace(s)
			if len(s) > needleLen {
				s = s[:needleLen]
			}
			if len(s) >= 16 {
				needles = append(needles, s)
			}
		}
	}
	web := filepath.Join(platformDir(t), "web", "src")
	if _, err := os.Stat(web); err != nil {
		t.Skipf("the pinned platform has no web sources at %s", web)
	}
	err := filepath.WalkDir(web, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !(strings.HasSuffix(path, ".ts") || strings.HasSuffix(path, ".tsx")) {
			return err
		}
		raw, err := os.ReadFile(path)
		for _, n := range needles {
			if strings.Contains(string(raw), n) {
				t.Errorf("%s copies the prompt %q: the platform reads it from the content", path, n)
			}
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}
