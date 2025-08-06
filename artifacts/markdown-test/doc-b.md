# Document B

[← Back to index](index.md) | [→ To Document A](doc-a.md)

## Installation

```bash
npm install
```

## Complex

This section has complex **markdown** scenarios:

### Mixed Formatting

- **[Bold link](doc-a.md)** followed by [normal link](index.md)
- *[Italic link](doc-a.md#section)* with **bold** text
- `code` then [link](index.md) then **bold**
- List item with • [link](doc-a.md) pattern

### Edge Cases

1. [Link 1](doc-a.md)[Link 2](index.md) - Adjacent links
2. **[Link](doc-a.md)****[Another](index.md)** - Adjacent bold links
3. [Multi
   line](doc-a.md) - Link with newline
4. **Bold with [nested link](index.md) inside**