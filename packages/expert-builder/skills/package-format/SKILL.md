---
name: package-format
description: Use whenever you draft, fix or explain an expert, skill pack, prompt pack or assist package - the exact files and manifest fields, requirements (skill packs, prompt packs, MCP tools), teams, scripts, where the save's limits are listed, and example packages.
---

# Package format

Every save validates the package against these rules; a draft that breaks
one is refused with an error saying what to fix. Before your first draft,
read `references/examples.md` beside this file: one complete, valid package
of each kind.

## Every package

A directory `drafts/<name>/` holding a `component.json` manifest: strict
JSON, unknown fields are refused.

- `kind`: `"bnbloc.expert.package/v1"`, `"bnbloc.skill.package/v1"`,
  `"bnbloc.prompt.package/v1"` or `"bnbloc.assist.package/v1"`.
- `name`: lower-case letters, digits and hyphens, starting with a letter or
  digit (`contract-reviewer`); the same as the directory.
- `version`: semantic version `major.minor.patch` without a leading `v`
  (`"1.0.0"`).
- `displayName` (one line) and `description`; optional `category`
  (lower-case letters, digits, hyphens), `tags` (lower-case letters,
  digits, spaces, `_`, `-`) and `icon` (one line, an icon name such as
  `scales`).
- `requires`: optional, see "Requirements".
- File and directory names use only `A-Z a-z 0-9 . _ -`.

## Expert (`bnbloc.expert.package/v1`)

```
drafts/<name>/
  component.json            + quickPrompts, welcome, team (all optional)
  INSTRUCTIONS.md           required, non-empty: the always-on role and method
  README.md                 optional, for people
  skills/<skill>/SKILL.md   optional embedded skills (+ references/ files)
```

- `quickPrompts`: `{ "title", "prompt" }` (title one line), or
  `{ "title"?, "ref": "<pack>/<prompt-id>" }` pointing at a prompt of a
  prompt pack named in `requires.prompts`.
- `welcome`: optional, shown when the expert is picked.
- Nothing else at the top level, and no `skills` or `prompts` field: the
  embedded skills are the `skills/<skill>/` directories.

## Skill pack (`bnbloc.skill.package/v1`)

```
drafts/<name>/
  component.json      + "skills": ["<skill>", ...]  (every skill directory, listed once)
  README.md           optional
  <skill>/SKILL.md    one directory per listed skill (+ references/ files)
```

- No `quickPrompts`, `welcome`, `prompts` or other top-level files.

## Prompt pack (`bnbloc.prompt.package/v1`)

```
drafts/<name>/
  component.json      + "prompts": [...]  (the prompts live here)
  README.md           optional
```

- No other files, and no `quickPrompts`, `welcome` or `skills`.
- Each prompt has:
  - `id`: lower-case letters, digits and hyphens, unique in the pack;
  - `title`: one line;
  - `body`: the prompt text, where `{{name}}` marks a variable;
  - `variables`: one `{ "name", "label"?, "default"?, "options"? }` per
    placeholder; `options` is a list to pick from, `default` one of them;
  - optional `tags`, as in the envelope;
  - optional `slash`: a composer command, `/` then lower-case letters,
    digits and hyphens, unique in the pack (`/notes`);
  - optional `starter: true`: offered among a room's starter prompts when
    the pack is on for the room. Mark only prompts that open a
    conversation well on their own, two or three per pack.
- Optional top-level `everyRoom: true`: the pack's starters are offered in
  every empty room, not only in rooms the pack is on for. Only for
  general-purpose starters the person asks to show everywhere.
- Every `{{name}}` is declared in `variables`, and every declared variable
  appears in the body. Keep bodies specific (who the answer is for, the
  format, the length) so the prompt is worth reusing.

## Assist (`bnbloc.assist.package/v1`)

```
drafts/<name>/
  component.json      + "output", "fields" (json only), "context", "gate" (optional)
  INSTRUCTIONS.md     required: the task's system prompt
  README.md           optional
```

- An assist runs on the deployment's assist model behind a button (the
  composer's "enhance prompt", a form's "Fill with AI"), not on an agent:
  no `requires`, `skills`, `prompts`, `quickPrompts` or other files.
- `output`: `"text"` (the reply replaces what the person typed) or `"json"`
  (the reply fills form fields).
- `fields` (json only, required there): `{ "name", "description"?,
  "type"?, "item"? }`; `name` is the form field's (`displayName`, `tags`);
  `type` is `"text"` (default), `"list"` (strings) or `"items"` (a list of
  objects whose own text fields `item` names, e.g. quick prompts
  `{title, prompt}`).
- `context`: what the server may add besides the typed text, any of
  `"room"` (its name, expert, skills and recent messages), `"artifact"`
  (a deliverable's title, type, file and recipe prompt), `"attachments"`
  (names and types), `"library"` (the experts and skill packs the person
  can use), `"clock"` (the current time in their timezone) and
  `"capabilities"` (the components listed to them, installed or not, the
  agents they can reach, and whether the room has other members) and
  `"conversation"` (the room's latest messages across its channels,
  bounded).
- `gate` (optional): `{ "question", "threshold" }`, one yes/no question
  for the decision model about the same context, and the lowest
  probability of yes (above 0) at which the task runs. A caller that runs the
  task on its own (after a turn) asks the gate first; below it nothing
  runs.
- The model gets the person's text and that context fenced as data, so
  `INSTRUCTIONS.md` says what to produce from them, in which language, and
  to return only the result.
- The name selects the task: an assist named like a default one
  (`enhance-prompt`) replaces it for its owner, or for the organization
  once published there. The screen that runs a default task reads its
  answer, so a replacement keeps the default's `output`, its `fields`
  with their types and item fields, and its `gate` (the save refuses one
  that does not): start from the default (`get_component`) and change
  the wording.

## SKILL.md (skill packs and experts)

YAML frontmatter whose `name` equals the skill's directory and whose
`description` says when to use the skill (the agent decides to load it from
that description alone), then the steps, checklists and examples.

## Requirements (`requires`)

- `skills`: skill packs the person can use, as `"name@^1"`.
- `prompts`: prompt packs, as `"name@^1"`; the expert's quick prompts may
  then `ref` their prompts instead of repeating them.
- `mcp`: MCP tools the agent needs, each by the name of an MCP tool in the
  person's Library (`list_components` with kind `mcp`), never a command or
  a URL: `[{ "name": "issue-tracker", "settings": ["apiToken"] }]`.
  `settings` names the keys of the tool's settings the package relies on;
  their values stay with the platform, the organization or the person,
  never in the package (a wrong key is refused with the tool's own list).
  Declare the permission the tool's transport needs: `"mcp.stdio"` for a
  local tool, `"mcp.http:<host>"` for a remote one (the save says which is
  missing).
- `permissions`: `"scripts"` when the package ships scripts (see
  "Scripts"), and the MCP permission above; otherwise omit.

## Teams (experts only)

A `team` block makes the expert a team. The agent the person addresses
with it picked is the lead: it wears this expert, whose `INSTRUCTIONS.md`
is the team's SOP, and hands parts of the task to the roles.

- `lead`: optional, the lead's title.
- `members`: the roles, each with
  - `role`: lower-case letters, digits and hyphens, unique, not the lead's
    title;
  - `expert`: the expert the role's agent wears, `"name@range"`, one the
    person can use and not itself a team (draft and save the role experts
    first when they do not exist yet);
  - `defaultAgent`: `requester` (the person's own agent), `platform` (a
    platform agent added to the room) or `member:<member id>` /
    `member:<member id>/<provider>`;
  - `description`: optional, what the role covers (the lead reads it).
- The SOP says how to split the task, what each role receives and returns,
  how to check and combine their work, and what to tell the person when a
  role cannot be staffed.

## Scripts

Prefer instructions and reference files. A file under a `scripts/` or
`bin/` directory, or with a script extension (`.sh`, `.py`, `.js`, `.ts`,
…), makes the package script-bearing: declare
`"requires": {"permissions": ["scripts"]}`. Script-bearing packages run in a
sandbox for other people and are saved only from the person's own agent;
add scripts only when the person asks for them.

## Limits

Every bound the save enforces (lengths, counts and sizes, per kind) is in
`content-limits.md` in bnbloc's room records in your workspace, generated
from the validator itself. Read it when a draft is long or large; this
skill states none, so it never disagrees with the save.
