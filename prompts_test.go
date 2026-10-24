// SPDX-FileCopyrightText: 2026 Bruno Guilhermo de Barros Valério
// SPDX-License-Identifier: AGPL-3.0-only

package voluspa

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/ouroboric-dev/gungnir/schema"
)

func prompt(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(softwareDir, "prompts", name))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

var fence = regexp.MustCompile("(?s)```[a-z]*\n(.*?)```")

// blocks returns the fenced code blocks of a prompt.
func blocks(text string) []string {
	var out []string
	for _, m := range fence.FindAllStringSubmatch(text, -1) {
		out = append(out, m[1])
	}
	return out
}

var planningPrompts = []string{"design.md", "spec.md", "conformance.md", "plan.md", "reuse-review.md"}

// The reviewers' prompts; a leaf realm's implementer prompt is the realm's own.
var buildPrompts = []string{"reviewer-spec.md", "reviewer-simplicity.md", "reviewer-quality.md"}

// sensible checks a prompt is a page or two and keeps the benchmark out of an agent's reach.
func sensible(t *testing.T, names []string) {
	t.Helper()
	for _, name := range names {
		text := prompt(t, name)
		if len(text) < 600 || len(text) > 12<<10 {
			t.Errorf("%s is %d bytes; a prompt is a page or two", name, len(text))
		}
		for _, bad := range []string{"huldra", "hidden suite", "ouroboric-private", "benchmark"} {
			if strings.Contains(strings.ToLower(text), bad) {
				t.Errorf("%s mentions %q; an agent that builds a product never learns what scores it", name, bad)
			}
		}
	}
}

func TestEveryPlanningPromptIsThereAndKeepsTheBenchmarkOutOfAgentsReach(t *testing.T) {
	sensible(t, planningPrompts)
}

func TestEveryBuildPromptIsThereAndKeepsTheBenchmarkOutOfAgentsReach(t *testing.T) {
	sensible(t, buildPrompts)
}

// A planning agent hands back typed output; no prompt asks for a file under out/ that Odin once collected, or for the date
// from a shell (the date is the input `date`).
func TestNoPromptAsksForAFileOrTheDate(t *testing.T) {
	for _, name := range append(append([]string{}, planningPrompts...), buildPrompts...) {
		text := prompt(t, name)
		for _, bad := range []string{"out/1-product", "out/2-engineering", "out/leaves.yaml", "out/conformance.json", "date +%F", "reviewed/", "docs/1-product", "docs/2-engineering", "docs/leaves.yaml"} {
			if strings.Contains(text, bad) {
				t.Errorf("%s holds %q; the inputs come in the prompt and the output goes back through submit", name, bad)
			}
		}
	}
}

// Each planning prompt names its contract, its inputs by name, and how to hand the output back. The field list and the
// schema are Odin's to append, so a prompt points at "Your answer" and does not repeat the schema.
// Odin renders an input as "## Input: <name>", so no prompt points at a section by another name.
func TestNoPromptNamesAStaleSection(t *testing.T) {
	for _, name := range append(append([]string{}, planningPrompts...), buildPrompts...) {
		text := prompt(t, name)
		for _, bad := range []string{`under "Brief"`, `"The change"`, `("Your task")`} {
			if strings.Contains(text, bad) {
				t.Errorf("%s holds %q; Odin renders each input as ## Input: <name>", name, bad)
			}
		}
	}
	for _, name := range planningPrompts {
		if !strings.Contains(prompt(t, name), "outside your workspace") {
			t.Errorf("%s lacks the workspace rule", name)
		}
	}
}

func TestEachPlanningPromptNamesItsContract(t *testing.T) {
	big := map[string]bool{"spec.md": true, "plan.md": true}
	for name, w := range map[string]struct {
		contract string
		inputs   []string
	}{
		"design.md":       {"ygg.design/v1", []string{"`brief`", "`decisions`", "`rejection`", "`date`"}},
		"spec.md":         {"ygg.spec/v1", []string{"`design`", "`decisions`", "`brief`", "`rejection`", "`refusal`", "`date`"}},
		"conformance.md":  {"ygg.conformance/v1", []string{"`spec`", "`design`"}},
		"plan.md":         {"ygg.plan/v1", []string{"`spec`", "`design`", "`decisions`", "`rejection`", "`refusal`", "`date`"}},
		"reuse-review.md": {"ygg.reuse-review/v1", []string{"`plan`", "`spec`", "`rejection`"}},
	} {
		text := prompt(t, name)
		for _, want := range append([]string{w.contract, "Your answer", "`submit`", "`blocked`"}, w.inputs...) {
			if !strings.Contains(text, want) {
				t.Errorf("%s does not mention %s", name, want)
			}
		}
		for _, large := range []string{"out/", "`check`"} {
			if has := strings.Contains(text, large); has != big[name] {
				t.Errorf("%s mentions %s: %v; only a large output is drafted in a file and checked", name, large, has)
			}
		}
		if big[name] && !strings.Contains(text, ".json") {
			t.Errorf("%s does not name the draft file", name)
		}
		if strings.Contains(text, `"$schema"`) || strings.Contains(text, `"additionalProperties"`) {
			t.Errorf("%s repeats the schema; Odin appends it", name)
		}
	}
	if !strings.Contains(prompt(t, "design.md"), "`q1`") {
		t.Error("design.md does not ask for open question ids q1, q2, ...")
	}
	spec := prompt(t, "spec.md")
	for _, want := range []string{"`decisions[].cites`", "`deviations[].cites`"} {
		if !strings.Contains(spec, want) {
			t.Errorf("spec.md does not tell the agent to cite the owner decisions in %s", want)
		}
	}
}

// What the conformance prompt shows as the answer is an answer Odin accepts.
func TestTheConformancePromptsExamplesAreVerdictsOdinReads(t *testing.T) {
	text := prompt(t, "conformance.md")
	for _, ex := range []string{`{"deviation": false}`, `{"deviation": true, "amendment": "<text>"}`} {
		if !strings.Contains(text, ex) {
			t.Fatalf("the prompt lacks the example %s", ex)
		}
		c, err := schema.ParseConformance([]byte(strings.ReplaceAll(ex, "<text>", "The spec adds offline mode.")))
		if err != nil {
			t.Errorf("%s: %v", ex, err)
		} else if c.Deviation != strings.Contains(ex, "true") {
			t.Errorf("%s was read as %+v", ex, c)
		}
	}
}

// The task the plan prompt shows is a task of the plan contract, and a ready one; the leaves file the spec prompt shows
// is a leaves file Odin reads.
func TestThePlanAndSpecPromptsExamplesAreDocumentsOdinReads(t *testing.T) {
	var task string
	for _, b := range blocks(prompt(t, "plan.md")) {
		if strings.Contains(b, `"named_test"`) {
			task = b
		}
	}
	if task == "" {
		t.Fatal("the plan prompt shows no task")
	}
	task = strings.NewReplacer("<slug>", "gallery", "<title>", "the API stores galleries", "<a thing that is true when the task is done>", "a gallery can be created", "<another>", "a gallery has a name",
		"<the test file and test name that proves it>", "gallery_test.go TestCreateGallery", "<behaviour, routes, fields, limits and errors>", "POST /galleries creates one.").Replace(task)
	doc := `{"title": "Gallery", "slug": "gallery", "goal": "Galleries.", "decisions": [], "risks": [], "tasks": [` + task + `]}`
	plan, err := schema.Decode[schema.Plan](schema.ContractPlan, []byte(doc))
	if err != nil || len(plan.Tasks) != 1 {
		t.Fatalf("Decode(plan) = %+v, %v", plan, err)
	}
	if err := schema.CheckReady(schema.FeaturesFromPlan(plan), []string{"repos/api"}); err != nil {
		t.Errorf("the example task is not ready: %v", err)
	}
	var leaves string
	for _, b := range blocks(prompt(t, "spec.md")) {
		if strings.Contains(b, "\nleaves:") || strings.HasPrefix(b, "leaves:") {
			leaves = b
		}
	}
	l, err := schema.ParseLeaves("leaves.yaml", []byte(leaves))
	if err != nil {
		t.Fatalf("ParseLeaves() = %v", err)
	}
	if got := strings.Join(l.Paths(), ","); got != "repos/api,repos/web" || l.Leaves["repos/web"].Consumes[0] != "repos/api" {
		t.Errorf("leaves = %v", l.Leaves)
	}
}

// The spec prompt offers the realms of the client gallery at the versions the leaves check compares.
func TestTheSpecPromptNamesTheGallerysRealmsAtTheirVersions(t *testing.T) {
	text := prompt(t, "spec.md")
	for _, want := range []string{"`midgard-react` 0.1.0", "`helheim-go` 0.1.0"} {
		if !strings.Contains(text, want) {
			t.Errorf("spec.md does not offer %s: a spec that names another version fails the leaves check", want)
		}
	}
}

func TestTheReviewersSayWhatToLookAtAndHowToNameFindings(t *testing.T) {
	for name, ids := range map[string]string{"reviewer-spec.md": "spec-1", "reviewer-simplicity.md": "simple-1", "reviewer-quality.md": "quality-1"} {
		text := prompt(t, name)
		for _, want := range []string{"read-only", "`change`", ids, "file:line", "**blocker**"} {
			if want == "**blocker**" && name == "reviewer-simplicity.md" {
				want = "**Must fix**"
			}
			if !strings.Contains(text, want) {
				t.Errorf("%s lacks %q", name, want)
			}
		}
		if strings.Contains(text, "git commit") {
			t.Errorf("%s talks of committing; a reviewer changes nothing", name)
		}
	}
	if !strings.Contains(prompt(t, "reviewer-simplicity.md"), "The skill below") || !strings.Contains(prompt(t, "reuse-review.md"), "The skill below") {
		t.Error("a prompt that relies on the vendored skill must say that the skill follows it")
	}
}

// The spec prompt names the runtime the product runs on, the leaves and the variables; the plan prompt
// keeps every task inside its leaf.
func TestPromptsNameTheRuntime(t *testing.T) {
	raw := prompt(t, "spec.md")
	spec := strings.Join(strings.Fields(raw), " ")
	for _, want := range []string{
		"one machine", "Postgres and an S3 store", "the factory's `start.sh`",
		"`repos/web` and `repos/api`, and no other", "The web leaf serves port 3000", "The API never serves the web app",
		"`/app/.data/`", "writes files only under `DATA_DIR`",
		"`PORT`, `HOST`, `DATABASE_URL`, `DATA_DIR`, `S3_ENDPOINT`, `S3_REGION`, `S3_BUCKET`, `S3_ACCESS_KEY_ID` and `S3_SECRET_ACCESS_KEY`",
		"`PORT`, `HOST`, `API_URL`, `S3_ENDPOINT` and `S3_BUCKET`",
		"`niflheim-local` 0.1.0 (the only runtime realm)", "`ygg-store`", "`us-east-1`",
	} {
		if !strings.Contains(spec, want) {
			t.Errorf("spec.md lacks %q", want)
		}
	}
	var leaves string
	for _, b := range blocks(raw) {
		if strings.Contains(b, "\nleaves:") || strings.HasPrefix(b, "leaves:") {
			leaves = b
		}
	}
	if !strings.Contains(leaves, "runtime: {realm: niflheim-local, realm_version: 0.1.0}") {
		t.Error("the leaves example has no runtime pin")
	}
	l, err := schema.ParseLeaves("leaves.yaml", []byte(leaves))
	if err != nil {
		t.Fatalf("ParseLeaves() = %v", err)
	}
	if l.Runtime == nil || l.Runtime.Realm != "niflheim-local" || l.Runtime.RealmVersion != "0.1.0" {
		t.Errorf("runtime = %+v", l.Runtime)
	}
	if !strings.Contains(spec, "Photo bytes and zips of photos go between the browser and the store, never through the API.") {
		t.Error("spec.md lacks the photo rule")
	}
	if !strings.Contains(spec, "`serve.mjs` does the forwarding, and no task builds it") {
		t.Error("spec.md does not say that serve.mjs forwards and no task builds it")
	}
	if strings.Contains(spec, "no server of its own") {
		t.Error("spec.md allows a product without repos/api")
	}
	plan := strings.Join(strings.Fields(prompt(t, "plan.md")), " ")
	for _, want := range []string{"No task writes any file outside its leaf.", "no task writes a `start.sh` anywhere, `/app/start.sh` included"} {
		if !strings.Contains(plan, want) {
			t.Errorf("plan.md lacks %q", want)
		}
	}
}

func TestReviewerNamesThePhotoRule(t *testing.T) {
	text := strings.Join(strings.Fields(prompt(t, "reviewer-spec.md")), " ")
	for _, want := range []string{"An API that streams a photo's bytes or a zip of photos breaks the rule that photo bytes never pass through the API (Yggdrasil spec §9.3): **blocker**."} {
		if !strings.Contains(text, want) {
			t.Errorf("reviewer-spec.md lacks %q", want)
		}
	}
}

// A reviewer has no shell: it reads the change in the section Odin puts in its prompt, not from git.
func TestTheReviewerPromptsReadTheChangeSectionAndRunNoGit(t *testing.T) {
	for _, name := range buildPrompts {
		text := prompt(t, name)
		if !strings.Contains(text, "your input `change`") {
			t.Errorf("%s does not name the input `change`", name)
		}
		for _, bad := range []string{"git diff", "git merge-base"} {
			if strings.Contains(text, bad) {
				t.Errorf("%s tells the reviewer to run %q; a reviewer has no shell", name, bad)
			}
		}
	}
}
