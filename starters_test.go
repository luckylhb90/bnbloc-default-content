package defaultcontent

import (
	"slices"
	"testing"
)

// The empty room's general-purpose starters are a prompt pack offered in
// every empty room (everyRoom): the platform's web shows the room's packs'
// starters, then those of the packs offered everywhere.
func TestStartersAreOfferedInEveryRoom(t *testing.T) {
	p := get(t, "starters")
	if p.Manifest.Kind != kindPrompt || !p.Manifest.EveryRoom {
		t.Fatalf("starters = %s, everyRoom %v", p.Manifest.Kind, p.Manifest.EveryRoom)
	}
	if len(p.Manifest.Prompts) < 3 {
		t.Fatalf("%d starters, want at least 3", len(p.Manifest.Prompts))
	}
	for _, pr := range p.Manifest.Prompts {
		if !pr.Starter {
			t.Errorf("prompt %s is not flagged starter", pr.ID)
		}
		if len(pr.Variables) > 0 {
			t.Errorf("starter %s asks for variables: a first message should need no form", pr.ID)
		}
		if pr.Files != nil || len(pr.Selection) > 0 {
			t.Errorf("room starter %s is scoped to files", pr.ID)
		}
	}
	// Only the general-purpose pack is offered everywhere.
	for name, q := range packages(t) {
		if q.Manifest.EveryRoom && name != "starters" {
			t.Errorf("%s is offered in every empty room", name)
		}
	}
}

// File suggestions are the same mechanism, scoped: the file prompts are
// starters with files or a selection kind, never neither, and need no form
// (bnbloc docs/prd/2026-10-14-file-prompts-and-compare.md rule 2).
func TestFilePromptsAreScopedStarters(t *testing.T) {
	selections, pdf := 0, 0
	for _, pr := range get(t, "file-prompts").Manifest.Prompts {
		if !pr.Starter || (pr.Files == nil && len(pr.Selection) == 0) {
			t.Errorf("file prompt %s: starter %v, files %v, selection %v", pr.ID, pr.Starter, pr.Files, pr.Selection)
		}
		if len(pr.Selection) > 0 {
			selections++
		}
		if len(pr.Variables) > 0 {
			t.Errorf("file prompt %s asks for variables: a suggestion should need no form", pr.ID)
		}
		if pr.Files != nil && slices.Contains(pr.Files.Extensions, ".pdf") && slices.Contains(pr.Selection, "text") {
			pdf++
		}
	}
	if selections == 0 {
		t.Error("no selection suggestions ship")
	}
	if pdf == 0 {
		t.Error("no PDF text selection prompts ship")
	}
}

// The per-domain suggestion packs (bnbloc
// docs/prd/2026-10-16-ask-ai-every-file.md rule 6) are scoped starters,
// and their selection prompts always name the files they are for: generic
// prompts are the fallback a scoped prompt replaces, so only file-prompts
// carries them.
func TestTypePromptPacksAreScoped(t *testing.T) {
	want := map[string][]string{
		"code-prompts":    {"text", "block"},
		"diagram-prompts": {"block", "text", "region"},
	}
	for name, kinds := range want {
		covered := map[string]bool{}
		for _, pr := range get(t, name).Manifest.Prompts {
			if !pr.Starter || pr.Files == nil {
				t.Errorf("%s/%s: starter %v, files %v: every prompt of a type pack is scoped to files", name, pr.ID, pr.Starter, pr.Files)
			}
			if len(pr.Variables) > 0 {
				t.Errorf("%s/%s asks for variables: a suggestion should need no form", name, pr.ID)
			}
			for _, k := range pr.Selection {
				covered[k] = true
			}
		}
		for _, k := range kinds {
			if !covered[k] {
				t.Errorf("%s offers nothing for a %q selection", name, k)
			}
		}
	}
}

// The media-skills pack requires the media-tools MCP component (the
// installable surface of the media capabilities), by name.
func TestMediaSkillsRequireMediaTools(t *testing.T) {
	p := get(t, "media-skills")
	var reqs []string
	for _, m := range p.Manifest.Requires.MCP {
		reqs = append(reqs, m.Name)
	}
	if p.Manifest.Kind != kindSkill || len(reqs) != 1 || reqs[0] != "media-tools" {
		t.Fatalf("media-skills = %s, requires.mcp %v", p.Manifest.Kind, reqs)
	}
}
