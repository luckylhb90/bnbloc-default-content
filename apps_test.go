package defaultcontent

import (
	"regexp"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/luckylhb90/bnbloc/component"
	"github.com/luckylhb90/bnbloc/core"
	"github.com/luckylhb90/bnbloc/services/gateway/appsdk"
)

// The interactive apps' content: the skill pack that teaches building an
// app on the SDK, the expert wearing it and the starter prompts. The SDK's
// contract is the platform's: its declarations (appsdk.Types), which every
// room's records hold (component.RoomAppSDKTypesFile); the skill points
// there and copies none of it.
const (
	appsSkillFile   = "interactive-apps/SKILL.md"
	appsExamples    = "interactive-apps/references/examples.md"
	appsSelfTest    = "interactive-apps/references/self-test.md"
	appsSkillMaxTk  = 1870 // below the 8 KiB a core without agentProfile inlines a skill body at
	appBuilderMaxTk = 700
)

var (
	tsCommentRE   = regexp.MustCompile(`(?s)/\*.*?\*/|//[^\n]*`)
	tsInterfaceRE = regexp.MustCompile(`(?s)interface (\w+)[^{\n]*\{\n(.*?)\n  \}`)
	tsMethodRE    = regexp.MustCompile(`(?m)^\s{4}(\w+)(?:<.*?>)?\(`)
	tsMemberRE    = regexp.MustCompile(`(?m)^\s{4}(\w+)\??:`)
	appCallRE     = regexp.MustCompile(`\bapp\.(\w+)\(`)
	collVarRE     = regexp.MustCompile(`(?:const|let|var)\s+(\w+)\s*=\s*app\.collection\(`)
	// statedCapRE is a cap of an app's data stated in prose: the
	// declarations own them.
	statedCapRE = regexp.MustCompile(`(?i)\d[\d,]*\s*(KiB|MiB|GiB)\s+(per|each)|\d[\d,k]*\s+documents|\d+\s+collections`)
)

// sdkDecl is what sdk.d.ts declares: the app's methods and members, and a
// collection's methods.
type sdkDecl struct {
	app, collection []string
}

func readSDKDecl(t *testing.T) sdkDecl {
	t.Helper()
	src := tsCommentRE.ReplaceAllString(string(appsdk.Types), "")
	var d sdkDecl
	for _, m := range tsInterfaceRE.FindAllStringSubmatch(src, -1) {
		var names []string
		for _, mm := range tsMethodRE.FindAllStringSubmatch(m[2], -1) {
			names = append(names, mm[1])
		}
		for _, mm := range tsMemberRE.FindAllStringSubmatch(m[2], -1) {
			names = append(names, mm[1])
		}
		switch m[1] {
		case "BnblocApp":
			d.app = names
		case "BnblocAppCollection":
			d.collection = names
		}
	}
	if len(d.app) == 0 || len(d.collection) == 0 {
		t.Fatalf("sdk.d.ts parsed to %+v: the parser no longer reads it", d)
	}
	return d
}

// The skill points at the SDK's declarations in the room's records and
// states no cap of its own; every SDK call in the skill and its references
// is one sdk.d.ts declares, so a renamed method fails here until the skill
// follows. Its delivery rules hold: the self-test before calling an app
// done, and the cover.
func TestInteractiveAppsSkillFollowsTheSDK(t *testing.T) {
	d := readSDKDecl(t)
	p := get(t, "interactive-apps")
	skill, examples, selfTest := p.file(t, appsSkillFile), p.file(t, appsExamples), p.file(t, appsSelfTest)
	for name, text := range map[string]string{appsSkillFile: skill, appsExamples: examples, appsSelfTest: selfTest} {
		for _, m := range appCallRE.FindAllStringSubmatch(text, -1) {
			if !slices.Contains(d.app, m[1]) {
				t.Errorf("%s calls app.%s, which the SDK does not have", name, m[1])
			}
		}
		for _, v := range collVarRE.FindAllStringSubmatch(text, -1) {
			re := regexp.MustCompile(`\b` + regexp.QuoteMeta(v[1]) + `\.(\w+)\(`)
			for _, m := range re.FindAllStringSubmatch(text, -1) {
				if !slices.Contains(d.collection, m[1]) {
					t.Errorf("%s calls %s.%s on a collection, which has no such method", name, v[1], m[1])
				}
			}
		}
	}
	for _, m := range statedCapRE.FindAllString(skill, -1) {
		t.Errorf("the skill states the cap %q: the SDK's declarations hold them", m)
	}
	types := strings.TrimPrefix(component.RoomAppSDKTypesFile, core.RoomContextDir)
	demo := strings.TrimPrefix(component.RoomAppSDKFile, core.RoomContextDir)
	limits := strings.TrimPrefix(component.RoomContentLimitsFile, core.RoomContextDir)
	says(t, "the interactive-apps skill", skill,
		"`"+types+"`", "`"+demo+"`", "`"+limits+"`",
		`<script src="~app/sdk.js"></script>`, "`{public: true}`", "{ifAbsent: true}", "`$inc`", "`subscribe`", "app_data", "`index.html`", "`dist/`",
		"`cover.png`", "headless browser", "never SVG",
		"no console error", "visible content", "writes through the SDK", "read back",
		"Never say an app is done or works", "`references/self-test.md`")
	says(t, "the self-test reference", selfTest, `"/~app/sdk.js"`, "pageerror", "innerText", "app.collection(", "never call the\napp done")
}

// The always-on parts stay small: the expert's instructions ride every
// request of its sessions, and a core without agentProfile inlines the
// skill's body into the first prompt (up to 8 KiB).
func TestAppContentAlwaysOnBudget(t *testing.T) {
	for _, c := range []struct {
		what, text string
		max        int
	}{
		{"the app builder's INSTRUCTIONS.md", get(t, "app-builder").file(t, "INSTRUCTIONS.md"), appBuilderMaxTk},
		{"the interactive-apps SKILL.md", get(t, "interactive-apps").file(t, appsSkillFile), appsSkillMaxTk},
	} {
		if tokens := len(c.text) / 4; tokens > c.max {
			t.Errorf("%s is about %d tokens, over its budget of %d: move examples to the skill's reference file", c.what, tokens, c.max)
		}
	}
}

// The App builder wears the interactive-apps skill pack and offers the app
// starters as its quick prompts, by name (whatever scope publishes them).
func TestAppBuilderRequiresItsSkillAndPrompts(t *testing.T) {
	b := get(t, "app-builder")
	m := b.Manifest
	if !slices.Contains(m.Requires.Skills, "interactive-apps@^1") || !slices.Contains(m.Requires.Prompts, "app-starters@^1") {
		t.Fatalf("the app builder requires %+v", m.Requires)
	}
	says(t, "the app builder", b.file(t, "INSTRUCTIONS.md"), "`interactive-apps`", "app_data")
	starters := get(t, "app-starters")
	ids := map[string]bool{}
	var slashes []string
	for _, p := range starters.Manifest.Prompts {
		ids[p.ID] = true
		slashes = append(slashes, p.Slash)
		if len(p.Variables) == 0 {
			t.Errorf("starter %s asks for nothing: each takes the app's specifics as variables", p.ID)
		}
	}
	for _, q := range m.QuickPrompts {
		pack, id, _ := strings.Cut(q.Ref, "/")
		if q.Ref != "" && (pack != "app-starters" || !ids[id]) {
			t.Errorf("quick prompt %q does not name an app starter", q.Ref)
		}
	}
	sort.Strings(slashes)
	if strings.Join(slashes, " ") != "/app-report /poll /signup /survey" {
		t.Fatalf("app starters' commands = %v", slashes)
	}
	// The interactive-apps skill pack's major version stays the one the
	// builder requires.
	if v := get(t, "interactive-apps").Manifest.Version; !strings.HasPrefix(v, "1.") {
		t.Errorf("interactive-apps %s no longer meets the builder's ^1", v)
	}
}
