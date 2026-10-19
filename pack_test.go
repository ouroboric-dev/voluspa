// SPDX-FileCopyrightText: 2026 Bruno Guilhermo de Barros Valério
// SPDX-License-Identifier: AGPL-3.0-only

package voluspa

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/ouroboric-dev/gungnir"
	"github.com/ouroboric-dev/gungnir/schema"
)

const softwareDir = "software"

func loadPack(t *testing.T) *schema.Pack {
	t.Helper()
	p, err := schema.LoadPack(softwareDir)
	if err != nil {
		t.Fatalf("LoadPack(%s) = %v", softwareDir, err)
	}
	return p
}

func loadGraph(t *testing.T) *schema.Graph {
	t.Helper()
	g, err := schema.LoadGraph(filepath.Join(softwareDir, "graph.yaml"))
	if err != nil {
		t.Fatalf("LoadGraph() = %v", err)
	}
	return g
}

func names(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, e := range entries {
		out = append(out, e.Name())
	}
	return out
}

func TestTheSoftwarePackHoldsAGraphAndItsPromptsOnly(t *testing.T) {
	p := loadPack(t)
	if p.Name != "software" || p.Version != "0.1.0" || p.Graph != "graph.yaml" {
		t.Errorf("pack = %+v", p)
	}
	if p.HoldsRecipe() {
		t.Error("the software pack holds a recipe (checks, leaf, leaves, sandbox or services); a graph pack holds none")
	}
	if g := loadGraph(t); g.Schema != 1 {
		t.Errorf("graph schema = %d, want 1", g.Schema)
	}
	if got := names(t, softwareDir); !slices.Equal(got, []string{"graph.yaml", "pack.yaml", "prompts"}) {
		t.Errorf("software/ holds %v, want graph.yaml, pack.yaml and prompts/", got)
	}
	want := []string{"conformance.md", "design.md", "plan.md", "reuse-review.md", "reviewer-quality.md", "reviewer-simplicity.md", "reviewer-spec.md", "spec.md"}
	if got := names(t, filepath.Join(softwareDir, "prompts")); !slices.Equal(got, want) {
		t.Errorf("software/prompts/ holds %v, want %v", got, want)
	}
}

func TestEveryFolderWithAPackIsAGraphPack(t *testing.T) {
	var graphs int
	for _, name := range names(t, ".") {
		if _, err := os.Stat(filepath.Join(name, "pack.yaml")); err != nil {
			continue
		}
		graphs++
		p, err := schema.LoadPack(name)
		if err != nil {
			t.Errorf("LoadPack(%s) = %v", name, err)
			continue
		}
		if p.Name != name || p.Graph == "" || p.HoldsRecipe() {
			t.Errorf("%s holds the pack %+v; a folder here is a graph pack named as the folder", name, p)
		}
	}
	if graphs == 0 {
		t.Error("no folder holds a pack")
	}
}

func TestTheGraphHasTheStagesOfSpecSection42InOrder(t *testing.T) {
	g := loadGraph(t)
	var ids []string
	for _, n := range g.Nodes {
		ids = append(ids, n.ID)
	}
	want := "design,design-gate,design-docs,spec,conformance,spec-gate,spec-docs,plan,reuse-review,plan-docs,reuse-docs,ready,leaves,plan-gate,tasks"
	if strings.Join(ids, ",") != want {
		t.Errorf("nodes = %v", ids)
	}
	by := map[string]schema.Node{}
	for _, n := range g.Nodes {
		by[n.ID] = n
	}
	if by["conformance"].Reviews != "spec" || by["spec-gate"].DeviationFrom != "conformance" || by["spec-gate"].Reruns != "spec" ||
		by["reuse-review"].Reviews != "plan" || by["ready"].BackTo != "plan" || by["plan-gate"].Reruns != "plan" {
		t.Errorf("the spec stage and the plan stage are not wired as spec §4.2 says: %+v", by)
	}
	if !slices.Contains(by["reuse-review"].Skills, "ponytail-review") {
		t.Error("the reuse review does not load the Ponytail skill")
	}
	// A planner works in a scratch workspace and publishes after its gate; none touches a leaf.
	for _, id := range []string{"design", "spec", "conformance", "plan", "reuse-review"} {
		if !by[id].Scratch || by[id].Repo != "" {
			t.Errorf("%s is not a scratch agent", id)
		}
	}
	for id, from := range map[string]string{"design-docs": "design", "spec-docs": "spec", "plan-docs": "plan", "reuse-docs": "reuse-review"} {
		if by[id].From != from || by[id].To != "docs" {
			t.Errorf("%s publishes %s to %q, want %s to docs", id, by[id].From, by[id].To, from)
		}
	}
}

// The acceptance check itself is defined by the leaf realm that has one (midgard-react), so the graph only names it.
func TestEveryTaskRunsTheLoopWithThreeReviewersThenTheGatesAndTheMoveWithAcceptance(t *testing.T) {
	g := loadGraph(t)
	body := g.Nodes[len(g.Nodes)-1].Body
	var ids []string
	for _, n := range body {
		ids = append(ids, n.ID)
	}
	if strings.Join(ids, ",") != "implement,feature-gate,merge-gate,land,pointer-gate,contract-gate,move" {
		t.Fatalf("the task shape = %v", ids)
	}
	impl := body[0]
	var rv []string
	for _, r := range impl.Reviewers {
		rv = append(rv, r.ID)
	}
	if impl.RealmFrom != schema.RealmLeaf || impl.Role != "implementer" || impl.Attempts != 3 || strings.Join(rv, ",") != "spec-compliance,simplicity,quality" {
		t.Errorf("implement = %+v; reviewers %v run in the spec's order (§9.5 rung 6)", impl, rv)
	}
	if !slices.Contains(impl.Reviewers[1].Skills, "ponytail-review") {
		t.Error("the simplicity reviewer does not load the Ponytail skill")
	}
	land, move := body[3], body[6]
	if land.BackTo != "implement" || move.BackTo != "implement" || !slices.Equal(move.Acceptance, []string{"accept"}) {
		t.Errorf("land %+v, move %+v: a refusal goes back to the implementer, and the move runs the acceptance check", land, move)
	}
}

func TestEveryPromptIsUsedAndEveryPromptTheGraphNamesIsThere(t *testing.T) {
	g := loadGraph(t)
	used := map[string]bool{}
	roles := map[string]bool{}
	exists := func(p string) error {
		_, err := os.Stat(filepath.Join(softwareDir, p))
		return err
	}
	for _, n := range g.All() {
		if n.Prompt != "" {
			used[filepath.Base(n.Prompt)] = true
			if err := exists(n.Prompt); err != nil {
				t.Errorf("node %s names the prompt %s: %v", n.ID, n.Prompt, err)
			}
		}
		if n.Role != "" {
			roles[n.Role] = true
		}
		for _, r := range n.Reviewers {
			used[filepath.Base(r.Prompt)] = true
			roles[r.Role] = true
			if err := exists(r.Prompt); err != nil {
				t.Errorf("reviewer %s names the prompt %s: %v", r.ID, r.Prompt, err)
			}
		}
	}
	for _, name := range names(t, filepath.Join(softwareDir, "prompts")) {
		if !used[name] {
			t.Errorf("prompts/%s is used by no node", name)
		}
	}
	for r := range roles {
		if !slices.Contains([]string{"planner", "implementer", "reviewer"}, r) {
			t.Errorf("the graph uses the role %q, which an operator's factory.yaml is not told to define", r)
		}
	}
}

// Spec §3.4: a move that changes a contract is the pointer-contract approval point. The gate judges the move
// that follows it, so it comes after the leaf has landed and before the move. The move also ships the routing
// indexes (spec §8): Odin builds REPOS.md's wiring block and .ygg/impact.json, and the shrink guard reads them.
func TestTheContractGateComesBeforeTheMoveAndTheMoveShipsTheRoutingIndexes(t *testing.T) {
	body := loadGraph(t).Nodes[len(loadGraph(t).Nodes)-1].Body
	by := map[string]int{}
	for i, n := range body {
		by[n.ID] = i
	}
	gate, move := body[by["contract-gate"]], body[by["move"]]
	if gate.Kind != schema.KindGate || gate.Gate != gungnir.GatePointerContract || by["contract-gate"] != by["move"]-1 || by["contract-gate"] < by["land"] {
		t.Errorf("contract-gate = %+v at %d; want the pointer-contract gate right before the move and after the land", gate, by["contract-gate"])
	}
	if !slices.Equal(move.Generated, []string{"REPOS.md", ".ygg/impact.json"}) {
		t.Errorf("the move generates %v; it must list both routing indexes so Odin builds them and the shrink guard reads them", move.Generated)
	}
	if loadPack(t).Routing != nil {
		t.Error("the pack sets a routing threshold; a project needs its bench first")
	}
}
