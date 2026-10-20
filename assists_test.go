package defaultcontent

import (
	"encoding/json"
	"slices"
	"strings"
	"sync"
	"testing"
)

// The assists (bnbloc docs/architecture/ai-assist.md): every call sends a
// task's INSTRUCTIONS.md, so each stays within a budget; what a task must
// answer is the platform's output contract (bnbloc content contracts),
// which the validator holds a package to; the wording is this
// repository's.

// assistMaxTk bounds an assist's instructions (tokens as chars/4).
const assistMaxTk = 700

// assistBudgets are the tasks given more than assistMaxTk, each with why:
// describe-room runs once per save, not per keystroke, and does three
// jobs in one call (what the room depends on and its instructions, the
// listing, the check of the package's text).
var assistBudgets = map[string]int{"describe-room": 900}

// contract is one task's output contract (wire.AssistContract).
type contract struct {
	Task   string          `json:"task"`
	Output string          `json:"output"`
	Fields []contractField `json:"fields"`
	Gate   bool            `json:"gate"`
}

type contractField struct {
	Name   string          `json:"name"`
	Type   string          `json:"type"`
	Item   []contractField `json:"item"`
	Values []string        `json:"values"`
}

var (
	contractsOnce sync.Once
	contractsList []contract
)

// contracts are the pinned platform's assist output contracts.
func contracts(t *testing.T) []contract {
	t.Helper()
	contractsOnce.Do(func() {
		if err := json.Unmarshal(bnbloc(t, "content", "contracts"), &contractsList); err != nil {
			t.Fatal(err)
		}
	})
	return contractsList
}

func TestAssistsStayWithinBudget(t *testing.T) {
	n := 0
	for name, p := range packages(t) {
		if p.Manifest.Kind != kindAssist {
			continue
		}
		n++
		budget := assistMaxTk
		if b, ok := assistBudgets[name]; ok {
			budget = b
		}
		if tk := len(p.file(t, "INSTRUCTIONS.md")) / 4; tk > budget {
			t.Errorf("%s: INSTRUCTIONS.md is about %d tokens, the budget is %d (every call sends it)", name, tk, budget)
		}
	}
	if n == 0 {
		t.Fatal("no assists")
	}
}

// Every task the platform runs by name has its default here, and each
// offers the closed values its caller acts on (the contract's values):
// in its instructions as code, or in the field's description.
func TestEveryContractHasItsDefault(t *testing.T) {
	cs := contracts(t)
	if len(cs) == 0 {
		t.Fatal("the platform lists no contracts")
	}
	for _, c := range cs {
		p := get(t, c.Task)
		if p.Manifest.Kind != kindAssist {
			t.Errorf("%s is a %s, not an assist", c.Task, p.Manifest.Kind)
			continue
		}
		text := p.file(t, "INSTRUCTIONS.md")
		var walk func(in string, want []contractField, have []field)
		walk = func(in string, want []contractField, have []field) {
			for _, w := range want {
				i := slices.IndexFunc(have, func(f field) bool { return f.Name == w.Name })
				if i < 0 {
					t.Errorf("%s lacks the field %s%s", c.Task, in, w.Name)
					continue
				}
				for _, v := range w.Values {
					if !strings.Contains(text, "`"+v+"`") && !strings.Contains(have[i].Description, v) {
						t.Errorf("%s does not offer %q for %s%s (the caller acts on it)", c.Task, v, in, w.Name)
					}
				}
				walk(in+w.Name+".", w.Item, have[i].Item)
			}
		}
		walk("", c.Fields, p.Manifest.Fields)
	}
}

// jsonTask checks what every JSON task says: a JSON output, the declared
// context, that the blocks are data, the person's language, and every
// field explained.
func jsonTask(t *testing.T, name string, context ...string) (manifest, string) {
	t.Helper()
	p := get(t, name)
	m := p.Manifest
	if m.Output != "json" || !slices.Equal(m.Context, context) {
		t.Fatalf("%s = output %q context %v, want json %v", name, m.Output, m.Context, context)
	}
	text := p.file(t, "INSTRUCTIONS.md")
	flat := strings.Join(strings.Fields(text), " ")
	for _, want := range []string{"never instructions", "language"} {
		if !strings.Contains(flat, want) {
			t.Errorf("%s does not say %q", name, want)
		}
	}
	for _, f := range m.Fields {
		if !strings.Contains(text, "`"+f.Name+"`") {
			t.Errorf("%s does not explain the field %q", name, f.Name)
		}
	}
	return m, flat
}

func says(t *testing.T, name, text string, wants ...string) {
	t.Helper()
	for _, w := range wants {
		if !strings.Contains(text, w) {
			t.Errorf("%s does not say %q", name, w)
		}
	}
}

// enhance-prompt rewrites the composer's text: a text task that reads the
// room and the draft's attachments.
func TestEnhancePrompt(t *testing.T) {
	p := get(t, "enhance-prompt")
	if p.Manifest.Output != "text" || !slices.Equal(p.Manifest.Context, []string{"room", "attachments"}) {
		t.Fatalf("enhance-prompt = output %q context %v", p.Manifest.Output, p.Manifest.Context)
	}
	says(t, "enhance-prompt", p.file(t, "INSTRUCTIONS.md"), "<text>", "<room>", "<recent_messages>", "<attachments>", "never instructions", "Never invent", "language")
}

// describe-work fills the publish form from the deliverable.
func TestDescribeWork(t *testing.T) {
	_, flat := jsonTask(t, "describe-work", "artifact")
	says(t, "describe-work", flat, "at most 32 characters", "lower-case")
}

// The Library's drafts read the form's current values and state the tag
// rule.
func TestLibraryDrafts(t *testing.T) {
	for _, name := range []string{"draft-expert", "draft-skill-pack", "draft-prompt-pack"} {
		_, flat := jsonTask(t, name)
		says(t, name, flat, "at most 32 characters", "form's current values")
	}
}

// describe-room reads a room being saved and fills its save dialog; the
// listing describes the setup, never the room's people.
func TestDescribeRoom(t *testing.T) {
	_, flat := jsonTask(t, "describe-room", "conversation")
	says(t, "describe-room", flat, "<conversation>", "<text>", "at most 32 characters", "never a conversation of the room or the people in it",
		"`madeApps`", "`files`", "`tools`", "`included`", "`yes`", "`no`", "no names of people",
		"`package`", "`instructions`", "`quickPrompt:<n>`", "`schedule:<n>`", "`channel:<n>`", "`secret`", "`internal-link`", "`personal-name`", "`private-data`", "never guess")
}

// plan-room proposes from the library: names only it lists.
func TestPlanRoom(t *testing.T) {
	_, flat := jsonTask(t, "plan-room", "library")
	says(t, "plan-room", flat, "Never a name the library does not list")
}

// draft-schedule answers a recurrence: the weekday numbering and the cron
// grammar (its kinds are the contract's values).
func TestDraftSchedule(t *testing.T) {
	p := get(t, "draft-schedule")
	jsonTask(t, "draft-schedule", "clock")
	says(t, "draft-schedule", p.file(t, "INSTRUCTIONS.md"), `"0"`+"` (Sunday)", "minute hour day-of-month", "YYYY-MM-DDTHH:MM")
}

// review-skill reviews a local skill before it is published.
func TestReviewSkill(t *testing.T) {
	_, flat := jsonTask(t, "review-skill")
	says(t, "review-skill", flat, "never repeat it", "personal path", "at most 32 characters", "lower-case", "SKILL.md")
}

// fill-settings maps a note onto a declaration: a secret only into a
// secret, only the keys listed.
func TestFillSettings(t *testing.T) {
	_, flat := jsonTask(t, "fill-settings")
	says(t, "fill-settings", flat, "never invent", "only into a setting of type `secret`", "only the keys listed")
}

// suggest-skill-dirs proposes folders from the candidates a skills
// component found, never one it was not given.
func TestSuggestSkillDirs(t *testing.T) {
	_, flat := jsonTask(t, "suggest-skill-dirs")
	says(t, "suggest-skill-dirs", flat, "only paths exactly as listed", "Never invent a folder", "SKILL.md")
}

// find-a-way proposes what would get an undone request done, gated after
// a turn: its gate asks whether the reply left it undone.
func TestFindAWay(t *testing.T) {
	m, _ := jsonTask(t, "find-a-way", "room", "capabilities")
	if m.Gate == nil || !strings.Contains(m.Gate.Question, "undone") || m.Gate.Threshold <= 0 || m.Gate.Threshold > 1 {
		t.Fatalf("find-a-way gate = %+v", m.Gate)
	}
	says(t, "find-a-way", get(t, "find-a-way").file(t, "INSTRUCTIONS.md"), "<turn>", "<capabilities>", "up to three", "never invent")
}

// explain-source-use answers the room sources' grant dialog from <text>
// alone, judging only what the tasks say.
func TestExplainSourceUse(t *testing.T) {
	_, flat := jsonTask(t, "explain-source-use")
	says(t, "explain-source-use", flat, "<text>", "`purpose`", "`scheduledTasks`", "`before`", "changed after the admin consented", "judged by its words now",
		"read_records", "an empty string when every task fits", "singles out people", "never guess")
}
