# Experience regression: what agents can do with and without this content

bnbloc docs/prd/2026-10-20-no-built-in-content.md §2.6 asks for a guard: a
set of real tasks run against a deployment with the default content and
one without, so a regression in what agents know shows as a failing task,
not a complaint. It needs a live deployment with a real agent and an
assist model, which CI does not have, so for now it is this checklist,
run by hand on QA after a platform release that changes what agents are
told (session guidance, room tools, served contract files) and after a
release of this repository. Record each run (date, platform version, this
repository's tag, the agent and its core) with the results in the release
notes.

## Set up

1. Two organizations on the same QA deployment, or two deployments:
   - **with**: `scripts/publish.sh --gateway <gw> --rollout auto` done;
   - **without**: every default package limited to the other
     organization (as a super admin, per package:
     `PUT /v1/platform/components/<name>/tenants?kind=<kind>` with
     `{"enabledTenants": ["<the with organization>"]}`), or a fresh
     deployment before the publish step.
2. In each, one member with their own agent on the desktop, an assist
   model configured, and the media tools and the browser component
   installed (they are components, not content: the same in both).
3. A new private room per task, no expert picked unless the task says so.

## Tasks

Each task is one message in the room. "Pass" is what the member sees.

| # | Task (the message) | With the default content | Without it |
|---|---|---|---|
| 1 | Pick **App builder**, then: "Make a lunch poll for Friday with three places; everyone votes once and sees the results live." | A site deliverable under the reply; voting twice from one account is refused; results update in a second browser; a cover image on the card; the reply says it tested the page in a headless browser | No App builder to pick: send the same message with no expert. Pass when the agent still builds a working poll from the room tools' briefs and the served SDK (`app-sdk/sdk.d.ts`); note what it got wrong |
| 2 | "Make an animated GIF guide of how to create a room, from the app's own screens." | The browser tool records the steps; a GIF deliverable | The same (the browser and media tools are components); note any difference |
| 3 | "Set up this room for our weekly release notes: a schedule every Friday at 4 pm that asks for the week's merged changes." then "Save this room as a component." | A disabled schedule created through room_setup; the save dialog filled by AI (describe-room) with the dependsOn checklist and findings | The schedule is created the same way; the save dialog has no AI fill (the button is hidden) and the manual form works |
| 4 | Ask for something the agent cannot do here ("Pull last month's payouts from our finance console") | The dead-end card under the reply proposes up to three fixes (find-a-way) | No card's AI proposals; the fixed fallback (tell your admins, rephrase) still shows |
| 5 | In the Library: **Create with AI** an expert, a skill pack and a prompt pack | The workshop opens with the Expert builder picked and a draft per kind; saving works | The workshop opens with a notice that no builder is installed; creating manually works |
| 6 | An empty room | The starter prompts show above the composer | No starters; the composer works |
| 7 | Every AI button: Enhance prompt, Fill with AI (work, room, expert, skill pack, prompt pack, schedule, settings), Plan a room, the skill review, the grant explanation in the console | Each runs and fills its form | Each button is hidden; each manual path works |

## Scripted part

What runs without a person:

```sh
# The packages validate against the deployment's platform version and
# every assist the web calls is installed for an organization:
go tool bnbloc content check --all packages
curl -sf -H "Authorization: Bearer $TOKEN" https://<gw>/v1/assist | jq -r '.tasks[].name' | sort
# expect the 13 names of `go tool bnbloc content contracts | jq -r '.[].task'`

# A long conversation, files and room tools still work across the platform
# release (bnbloc AGENTS.md "Releases replay real use"):
bnbloc smoke --scenario long-task
```

Automating tasks 1 to 6 needs a driven agent and a judge of the result;
until that exists, a failed row here is a release blocker for the side
that changed.
