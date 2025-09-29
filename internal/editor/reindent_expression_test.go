package editor

import (
	"strings"
	"testing"

	"veda/internal/indent"
)

func setBufferPath(te *TestEditor, path string) {
	buf := te.CurrentBuffer()
	oldPath := buf.FilePath
	buf.FilePath = path
	buf.RealPath = path
	if oldPath != path {
		delete(te.buffers, oldPath)
		te.buffers[path] = buf
		for i, p := range te.bufferOrder {
			if p == oldPath {
				te.bufferOrder[i] = path
				break
			}
		}
	}
}

func TestReindentExpression_FixesMisalignedInnerBlock(t *testing.T) {
	content := strings.Join([]string{
		"int main() {",
		"    if (x) {",
		"    foo();",
		"        }",
		"}",
	}, "\n")

	te := NewTestEditorWithContent(80, 24, content)
	setBufferPath(te, "test1.c")
	te.SetCursorPosition(4, 2)

	if err := te.ExecuteCommand("reindent-expression"); err != nil {
		t.Fatalf("reindent-expression failed: %v", err)
	}

	got := te.GetBufferContent()
	want := strings.Join([]string{
		"int main() {",
		"    if (x) {",
		"        foo();",
		"    }",
		"}",
	}, "\n")
	if got != want {
		t.Fatalf("unexpected buffer after reindent-expression.\nGot:\n%q\nWant:\n%q", got, want)
	}
}

func TestReindentExpression_MultilineConditionStaysAligned(t *testing.T) {
	content := strings.Join([]string{
		"void",
		"qux_override(void)",
		"{",
		"    if (FLAG_ZERO ||",
		"        mode_kind == MODE_PRIMARY ||",
		"        mode_kind == MODE_TRANSITION)",
		"    {",
		"        if (ctx.link_active) {",
		"            trace(LOG_INFO,",
		"                  \"placeholder\",",
		"                  mode_label(mode_kind));",
		"            ctx.link_active = FLAG_ZERO;",
		"        }",
		"    }",
		"}",
	}, "\n")

	te := NewTestEditorWithContent(80, 24, content)
	setBufferPath(te, "test2.c")
	te.SetCursorPosition(0, 3)

	before := te.GetBufferContent()
	if err := te.ExecuteCommand("reindent-expression"); err != nil {
		t.Fatalf("reindent-expression failed: %v", err)
	}
	after := te.GetBufferContent()
	if after != before {
		t.Fatalf("expected content to remain unchanged, but got:\n%s", after)
	}
}

func TestReindentExpression_RespectsOuterComments(t *testing.T) {
	content := strings.Join([]string{
		"int sum(int a, int b) {",
		"    return a + b;",
		"}",
		"    // helper summary lives outside the function",
	}, "\n")

	te := NewTestEditorWithContent(80, 24, content)
	setBufferPath(te, "test3.c")
	te.SetCursorPosition(4, 1)

	if err := te.ExecuteCommand("reindent-expression"); err != nil {
		t.Fatalf("reindent-expression failed: %v", err)
	}

	got := te.GetBufferContent()
	if got != content {
		t.Fatalf("trailing comment was modified.\nGot:\n%q\nWant:\n%q", got, content)
	}
}

func TestReindentExpression_ReflowsCallArguments(t *testing.T) {
	content := strings.Join([]string{
		"if outer {",
		"    foo(",
		"        a,",
		"    b,",
		"    c,",
		"    )",
		"}",
	}, "\n") + "\n"

	te := NewTestEditorWithContent(80, 24, content)
	setBufferPath(te, "test4.c")
	te.SetCursorPosition(0, 2)

	if err := te.ExecuteCommand("reindent-expression"); err != nil {
		t.Fatalf("reindent-expression failed: %v", err)
	}

	want := strings.Join([]string{
		"if outer {",
		"    foo(",
		"        a,",
		"        b,",
		"        c,",
		"    )",
		"}",
	}, "\n") + "\n"

	if got := te.GetBufferContent(); got != want {
		t.Fatalf("unexpected content after reindent-expression.\nGot:\n%s\nWant:\n%s", got, want)
	}
}

func TestReindentExpression_ReducesIndentAfterBlockExit(t *testing.T) {
	content := strings.Join([]string{
		"if (foo) {",
		"    baz();",
		"}",
		"    bar();",
	}, "\n") + "\n"

	te := NewTestEditorWithContent(80, 24, content)
	setBufferPath(te, "move_down_case.c")
	te.SetCursorPosition(0, 3)

	lang := indent.NewCLanguage()
	tokens, err := lang.Tokenize([]rune(content))
	if err != nil {
		t.Fatalf("tokenize failed: %v", err)
	}
	annotated, err := indent.NewAnalyzer(tokens, lang.Rules()).Analyze()
	if err != nil {
		t.Fatalf("analyze failed: %v", err)
	}
	line0 := make(map[int][2]int)
	for _, at := range annotated {
		if at.Token.Kind == indent.TokenWhitespace || at.Token.Kind == indent.TokenNewline {
			continue
		}
		if _, ok := line0[at.Token.Start.Line]; !ok {
			line0[at.Token.Start.Line] = [2]int{at.IndentLevel, at.ContinuationLevel}
		}
	}

	t.Logf("lineCount before: %d", te.CurrentBuffer().LineCount)

	if err := te.ExecuteCommand("reindent-expression"); err != nil {
		t.Fatalf("reindent-expression failed: %v", err)
	}

	want := strings.Join([]string{
		"if (foo) {",
		"    baz();",
		"}",
		"bar();",
	}, "\n") + "\n"

	if got := te.GetBufferContent(); got != want {
		t.Fatalf("unexpected buffer after reindent-expression.\nGot:\n%s\nWant:\n%s\nindent map: %+v", got, want, line0)
	}
}

func TestReindentExpression_DoesNotReindentWholeFunction(t *testing.T) {
	lines := []string{
		"static void fn_scramble_alpha(void) {",
		"",
		"    struct flux_bundle *bundle = mkflux_bundle(",
		"        /*trace_enabled=*/1, /*scale_factor*/1, /*skip_bootstrap*/1,",
		"        DEFAULT_FLUX_CAPACITY);",
		"",
		"    flux_region_token_t token = (__force flux_region_token_t)0x1;",
		"    flux_bundle_stage_region(bundle, token, PHASE_RED);",
		"",
		"    struct shard_table *tab = mkshard_table(NSEGMENTS, bundle->io_lane, bundle->cfg_core->table, bundle->phase,",
		"        FLUX_TABLE_TUNING_DELTA,",
		"        /*ui_shift=*/0u, /*ui_online=*/SIG_FALSE,",
		"        /*cooldown_timeout=*/60 * ONE_SECOND_TICKS,",
		"        bundle->signal_stats);",
		"",
		"    CHECK(DEBUG_flux_region_count(bundle->tab) == 1);",
		"",
		"    tab_spin_up(tab);",
		"",
		"    uint64_t epoch;",
		"    struct ipv4_endpoint endpoint;",
		"    endpoint = tab_find_primary(tab, token, flux_deadline_short(), &epoch);",
		"    ASSERT(!ipv4_endpoint_eq(endpoint, IPV4_ENDPOINT_ZERO));",
		"",
		"    // record outcome",
		"    tickns_t now = tickns();",
		"    tab_record_outcome(bundle->tab, token, epoch, endpoint, SIG_TRUE, now);",
		"    const struct endpoint_health health = DEBUG_tab_lookup(bundle->tab, endpoint);",
		"    ASSERT(health.last_seen==now);",
		"    ASSERT(health.failure_count==0);",
		"    /* First time we report a timeout we keep failure_count at zero.",
		"       The second timeout increments failure_count. */",
		"    tickns_t now2 = now + ONE_SECOND_TICKS_DEF;",
		"    tab_record_outcome(bundle->tab, token, epoch, endpoint, SIG_FALSE, now2);",
		"    const struct endpoint_health health_after = DEBUG_tab_lookup(bundle->tab, endpoint);",
		"    ASSERT(health_after.failure_count==0);",
		"    tickns_t now3 = now2 + ONE_SECOND_TICKS_DEF;",
		"    tab_record_outcome(bundle->tab, token, epoch, endpoint, SIG_FALSE, now3);",
		"    const struct endpoint_health health_final = DEBUG_tab_lookup(bundle->tab, endpoint);",
		"    ASSERT(health_final.failure_count==1);",
		"    tab_spin_down(tab);",
		"    tab = tab_destroy(tab);",
		"    bundle = flux_bundle_destroy(bundle);",
		"}",
	}
	content := strings.Join(lines, "\n") + "\n"

	te := NewTestEditorWithContent(120, 40, content)
	setBufferPath(te, "scramble_outcome.c")
	te.SetCursorPosition(8, 9)

	before := te.GetBufferContent()
	if err := te.ExecuteCommand("reindent-expression"); err != nil {
		t.Fatalf("reindent-expression failed: %v", err)
	}
	after := te.GetBufferContent()
	beforeLines := strings.Split(before, "\n")
	afterLines := strings.Split(after, "\n")
	if len(beforeLines) != len(afterLines) {
		t.Fatalf("unexpected line count change. before=%d after=%d", len(beforeLines), len(afterLines))
	}
	allowed := map[int]struct{}{3: {}, 4: {}, 10: {}, 11: {}, 12: {}, 13: {}}
	var changed []int
	for i := range beforeLines {
		if beforeLines[i] != afterLines[i] {
			changed = append(changed, i)
			if _, ok := allowed[i]; !ok {
				t.Fatalf("line %d changed unexpectedly.\nBefore: %q\nAfter:  %q", i, beforeLines[i], afterLines[i])
			}
		}
	}
	if len(changed) == 0 {
		t.Fatalf("expected targeted expression lines to change")
	}
}
