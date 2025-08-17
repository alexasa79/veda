- Terminal wide-character support (wcwidth/grapheme-aware)

  - Problem: Lines containing wide characters (e.g., emoji like 🐟 or CJK) can shift background/rulers by one column if we assume width 1. Terminals render many of these as width 2.
  - Goal: Use a proper wcwidth implementation so our emulator, renderer, wrapping, and cursor math all agree with the terminal’s notion of display columns.
  - Plan:
    - Integrate a wcwidth/grapheme library (e.g., github.com/mattn/go-runewidth) across:
      - TerminalEmulator.Write: advance by RuneWidth; mark trailing cell for width 2
      - Renderer (terminal path and normal files): advance x by RuneWidth and paint trailing cell for width 2
      - Wrapping/visual width/byte position conversions: base all on RuneWidth (and tabs)
      - Ruler checks: compare against visual columns (sum of RuneWidth), not bytes/len
    - Tests:
      - Single emoji file: rulers align, cursor after emoji
      - Fish prompt line with emoji: rulers align at configured columns
      - Mixed CJK, emoji, tabs; wrapped lines
  - Notes:
    - Consider Variation Selector 15/16 edge cases
    - Ensure consistent handling in both emulator and renderer paths

