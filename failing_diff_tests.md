# Failing Diff-Related Tests

## Summary
3 test functions are currently failing in the diff-related test suite.

## Failing Tests

### 1. TestDiffRendering (`internal/editor/diff_rendering_test.go`)
**Failure Location**: Subtest `diff_line_structure_verification`

**Issue**: After `detectModifications()` is applied, the LEFT buffer (HEAD) contains:
- Modified lines (type=3) 
- Added lines (type=1)

**Expected**: LEFT buffer should only contain:
- Context lines (type=0)
- Deleted lines (type=2)
- NO Modified or Added lines

**Error Messages**:
- Line 269: "Left side should not have added lines after fixing asymmetric highlighting"
- Line 272: "Left side should not have modified lines after fixing asymmetric highlighting"

**Root Cause**: The `detectModifications()` function converts delete+add pairs into modified lines, which violates the asymmetric highlighting principle where the left side should only show what was removed or unchanged.

---

### 2. TestWordDiffAlgorithm (`internal/editor/word_diff_test.go`)
**Failure Location**: Subtest `multiple_changes`

**Issue**: When comparing "The quick brown fox" → "The slow brown cat", the algorithm returns:
- 1 range: [4, 18) = "slow brown cat"

**Expected**: 2 separate ranges for the two word changes:
- Range for "quick" → "slow"
- Range for "fox" → "cat"

**Error Message**:
- Line 237: "Expected 2 ranges, got 1 for left=\"The quick brown fox\", right=\"The slow brown cat\""

**Root Cause**: The word diff algorithm is merging multiple changes into a single range instead of identifying individual word-level changes.

---

### 3. TestWordLevelDiffHighlighting (`internal/editor/word_diff_test.go`)
**Failure Location**: Subtest `word_level_highlighting_in_current_buffer`

**Issue**: Character at position 15 (a space character) is not highlighted when it should be.

**Test Case**: "Hello world, this is a test" → "Hello universe, this is a demo"
- Word range: [6, 30) = "universe, this is a demo"
- Position 15 (" ") should be highlighted but has normal background color

**Error Message**:
- Line 102: "Line 0, pos 15 (\" \"): expected highlighted background 23351326, got 18554142"

**Root Cause**: The rendering logic is not correctly applying word-level diff highlighting to all characters within the specified range, possibly skipping space characters.

---

## Test Files Analyzed
All diff-related test files were checked:
- `diff_alignment_test.go` ✅
- `diff_buffer_close_test.go` ✅  
- `diff_cursor_sync_test.go` ✅
- `diff_highlighting_test.go` ✅
- `diff_line_count_test.go` ✅
- `diff_line_numbers_test.go` ✅
- `diff_rendering_test.go` ❌
- `diff_syntax_highlighting_test.go` ✅
- `diff_view_width_test.go` ✅
- `diff_window_dimming_test.go` ✅
- `word_diff_debug_test.go` ✅
- `word_diff_test.go` ❌

## Recommendations for Fixes

1. **TestDiffRendering**: Modify `detectModifications()` to preserve asymmetric highlighting by not converting delete+add pairs to modifications when asymmetric mode is desired.

2. **TestWordDiffAlgorithm**: Update the word diff algorithm to identify individual word changes rather than merging them into a single contiguous range.

3. **TestWordLevelDiffHighlighting**: Fix the rendering logic to ensure all characters within word diff ranges are properly highlighted, including whitespace characters.