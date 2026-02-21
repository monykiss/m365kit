package docx

import (
	"strings"
	"testing"
)

func TestMyersDiffIdentical(t *testing.T) {
	lines := []string{"alpha", "beta", "gamma"}
	ops := myersDiff(lines, lines)

	for _, op := range ops {
		if op.Op != "=" {
			t.Errorf("expected all '=' ops for identical input, got %q for %q", op.Op, op.Text)
		}
	}
	if len(ops) != 3 {
		t.Errorf("expected 3 ops, got %d", len(ops))
	}
}

func TestMyersDiffInsertion(t *testing.T) {
	a := []string{"alpha", "gamma"}
	b := []string{"alpha", "beta", "gamma"}
	ops := myersDiff(a, b)

	inserts := 0
	for _, op := range ops {
		if op.Op == "+" {
			inserts++
			if op.Text != "beta" {
				t.Errorf("expected inserted text 'beta', got %q", op.Text)
			}
		}
	}
	if inserts != 1 {
		t.Errorf("expected 1 insertion, got %d", inserts)
	}
}

func TestMyersDiffDeletion(t *testing.T) {
	a := []string{"alpha", "beta", "gamma"}
	b := []string{"alpha", "gamma"}
	ops := myersDiff(a, b)

	deletes := 0
	for _, op := range ops {
		if op.Op == "-" {
			deletes++
			if op.Text != "beta" {
				t.Errorf("expected deleted text 'beta', got %q", op.Text)
			}
		}
	}
	if deletes != 1 {
		t.Errorf("expected 1 deletion, got %d", deletes)
	}
}

func TestMyersDiffEmpty(t *testing.T) {
	a := []string{}
	b := []string{"alpha", "beta"}
	ops := myersDiff(a, b)

	for _, op := range ops {
		if op.Op != "+" {
			t.Errorf("expected all '+' ops when original is empty, got %q", op.Op)
		}
	}
	if len(ops) != 2 {
		t.Errorf("expected 2 ops, got %d", len(ops))
	}
}

func TestMyersDiffAllDeleted(t *testing.T) {
	a := []string{"alpha", "beta"}
	b := []string{}
	ops := myersDiff(a, b)

	for _, op := range ops {
		if op.Op != "-" {
			t.Errorf("expected all '-' ops when revised is empty, got %q", op.Op)
		}
	}
	if len(ops) != 2 {
		t.Errorf("expected 2 ops, got %d", len(ops))
	}
}

func TestMyersDiffDeterministic(t *testing.T) {
	a := []string{"the", "quick", "brown", "fox"}
	b := []string{"the", "slow", "brown", "bear"}

	ops1 := myersDiff(a, b)
	ops2 := myersDiff(a, b)

	if len(ops1) != len(ops2) {
		t.Fatalf("non-deterministic: %d vs %d ops", len(ops1), len(ops2))
	}
	for i := range ops1 {
		if ops1[i].Op != ops2[i].Op || ops1[i].Text != ops2[i].Text {
			t.Errorf("non-deterministic at %d: %v vs %v", i, ops1[i], ops2[i])
		}
	}
}

func TestDiffParagraphsIdentical(t *testing.T) {
	paras := []string{"Hello world", "This is a test", "Goodbye"}
	result := DiffParagraphs(paras, paras, "a.docx", "b.docx", 3)

	if result.Insertions != 0 {
		t.Errorf("expected 0 insertions, got %d", result.Insertions)
	}
	if result.Deletions != 0 {
		t.Errorf("expected 0 deletions, got %d", result.Deletions)
	}
	if result.Unchanged != 3 {
		t.Errorf("expected 3 unchanged, got %d", result.Unchanged)
	}
	if len(result.Hunks) != 0 {
		t.Errorf("expected 0 hunks for identical docs, got %d", len(result.Hunks))
	}
}

func TestDiffParagraphsWithAddition(t *testing.T) {
	orig := []string{"intro", "conclusion"}
	rev := []string{"intro", "new paragraph", "conclusion"}
	result := DiffParagraphs(orig, rev, "orig.docx", "rev.docx", 3)

	if result.Insertions != 1 {
		t.Errorf("expected 1 insertion, got %d", result.Insertions)
	}
	if result.Deletions != 0 {
		t.Errorf("expected 0 deletions, got %d", result.Deletions)
	}
}

func TestDiffParagraphsWithDeletion(t *testing.T) {
	orig := []string{"intro", "middle", "conclusion"}
	rev := []string{"intro", "conclusion"}
	result := DiffParagraphs(orig, rev, "orig.docx", "rev.docx", 3)

	if result.Insertions != 0 {
		t.Errorf("expected 0 insertions, got %d", result.Insertions)
	}
	if result.Deletions != 1 {
		t.Errorf("expected 1 deletion, got %d", result.Deletions)
	}
}

func TestDiffParagraphsCompletelyReplaced(t *testing.T) {
	orig := []string{"old1", "old2", "old3"}
	rev := []string{"new1", "new2"}
	result := DiffParagraphs(orig, rev, "orig.docx", "rev.docx", 3)

	if result.Insertions+result.Deletions == 0 {
		t.Error("expected some changes for completely replaced content")
	}
	if result.Insertions != 2 {
		t.Errorf("expected 2 insertions, got %d", result.Insertions)
	}
	if result.Deletions != 3 {
		t.Errorf("expected 3 deletions, got %d", result.Deletions)
	}
}

func TestHunkHeaderFormat(t *testing.T) {
	orig := []string{"alpha", "beta", "gamma"}
	rev := []string{"alpha", "BETA", "gamma"}
	result := DiffParagraphs(orig, rev, "a.docx", "b.docx", 1)

	if len(result.Hunks) == 0 {
		t.Fatal("expected at least one hunk")
	}

	header := result.Hunks[0].Header
	if !strings.HasPrefix(header, "@@ ") || !strings.HasSuffix(header, " @@") {
		t.Errorf("hunk header format wrong: %q", header)
	}
	if !strings.Contains(header, "-") || !strings.Contains(header, "+") {
		t.Errorf("hunk header missing -/+: %q", header)
	}
}

func TestDiffFormatUnified(t *testing.T) {
	orig := []string{"intro", "middle", "conclusion"}
	rev := []string{"intro", "new middle", "conclusion"}
	result := DiffParagraphs(orig, rev, "orig.docx", "rev.docx", 1)

	output := result.FormatUnified(false)

	if !strings.Contains(output, "--- orig.docx") {
		t.Error("missing original file header")
	}
	if !strings.Contains(output, "+++ rev.docx") {
		t.Error("missing revised file header")
	}
	if !strings.Contains(output, "- middle") {
		t.Error("missing deletion line")
	}
	if !strings.Contains(output, "+ new middle") {
		t.Error("missing insertion line")
	}
}

func TestDiffStats(t *testing.T) {
	result := &DiffResult{
		Insertions: 2,
		Deletions:  1,
		Unchanged:  10,
	}
	stats := result.Stats()
	if stats != "2 insertions, 1 deletions, 10 unchanged" {
		t.Errorf("unexpected stats: %q", stats)
	}
}
