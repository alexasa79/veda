# Markdown table preview

Normal mode aligns these tables without changing their source. Press i to see
the original Markdown, then Escape to restore the aligned display.

## Opening moves

| Opening move in ATR14 units | Positions | Profit | Profit per $10,000 traded | Win rate |
|---|---:|---:|---:|---:|
| Q1: 0.0002 through 0.1233 | 977 | $2,279.01 | $9.07 | 29.0% |
| Q2: 0.1235 through 0.2616 | 976 | $3,220.37 | $12.78 | 29.8% |
| Q3: 0.2621 through 0.4676 | 976 | $4,541.87 | $17.94 | 28.6% |
| Q4: 0.4677 through 2.7823 | 976 | $5,654.47 | $23.32 | 24.4% |

Narrow the terminal and move right through a table row to see horizontal
scrolling. Table rows stay on one screen line in normal mode.

## Alignment and copying

|Label|Center|Positions|
|:---|:---:|---:|
|Q1|x|977|
|Longer label|middle|2|

Use v to select text and y to copy it. The cursor skips display-only padding;
copying uses the original text, including original pipes and whitespace when
the selection crosses cells.

Alt+Q reflows ordinary paragraphs. Inside a table it rewrites the Markdown
source with aligned columns; undo restores the original source.
