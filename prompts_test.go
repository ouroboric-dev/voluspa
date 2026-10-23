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

func TestThePlanningPromptsSayWhereTheirDocumentsGo(t *testing.T) {
	for name, paths := range map[string][]string{
		"design.md":       {"out/1-product/design/<date>-<slug>.md"},
		"spec.md":         {"out/2-engineering/specs/<date>-<slug>.md", "out/leaves.yaml", "docs/1-product/design/"},
		"conformance.md":  {"out/conformance.json", "reviewed/2-engineering/specs/", "reviewed/leaves.yaml", "docs/1-product/design/"},
		"plan.md":         {"out/2-engineering/plans/<date>-<slug>/", "docs/leaves.yaml", "docs/2-engineering/specs/"},
		"reuse-review.md": {"out/2-engineering/reviews/<date>-<slug>-reuse-review.md", "reviewed/2-engineering/plans/"},
	} {
		text := prompt(t, name)
		for _, p := range paths {
			if !strings.Contains(text, p) {
				t.Errorf("%s does not mention %s", name, p)
			}
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

// The task the plan prompt shows is a task Odin reads, and a ready one; the leaves file the spec prompt shows
// is a leaves file Odin reads.
func TestThePlanAndSpecPromptsExamplesAreFilesOdinReads(t *testing.T) {
	var task string
	for _, b := range blocks(prompt(t, "plan.md")) {
		if strings.Contains(b, "**Named test:**") {
			task = b
		}
	}
	if task == "" {
		t.Fatal("the plan prompt shows no task")
	}
	task = strings.NewReplacer("<plan>", "gallery", "<title>", "the API stores galleries", "<a thing that is true when the task is done>", "a gallery can be created", "<another>", "a gallery has a name",
		"<the test file and test name that proves it>", "gallery_test.go TestCreateGallery").Replace(task)
	fs, err := schema.ParseTasks("tasks.md", []byte(task))
	if err != nil || len(fs) != 1 {
		t.Fatalf("ParseTasks() = %v, %v", fs, err)
	}
	if err := schema.CheckReady(fs, []string{"repos/api"}); err != nil {
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
		for _, want := range []string{"read-only", `"The change"`, ids, "file:line", "**blocker**"} {
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
		if !strings.Contains(text, `"The change"`) {
			t.Errorf("%s does not name the section \"The change\"", name)
		}
		for _, bad := range []string{"git diff", "git merge-base"} {
			if strings.Contains(text, bad) {
				t.Errorf("%s tells the reviewer to run %q; a reviewer has no shell", name, bad)
			}
		}
	}
}
