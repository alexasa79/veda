# Diagram Language

This language describes diagrams using boxes, connections, and nesting. It is
intended for diagram code blocks in Markdown. The source is shown while editing
and rendered as a diagram in normal mode.

## Boxes and connections

An identifier refers to a box. Identifiers name boxes but are not displayed as
box contents.

```text
a -> b
```

This creates two equally sized empty boxes and draws an arrow from `a` to `b`.
A connection can also declare a box's text:

```text
a["hello world"] -> b
```

A box can be declared separately from its connections:

```text
a["hello world"]
a -> b
a -> c
```

Every occurrence of `a` refers to the same box, so its text persists across all
three statements.

## Nesting

Brackets containing diagram expressions place those boxes inside the outer
box:

```text
a[b -> c]
```

This creates a container box `a`, with connected boxes `b` and `c` inside it.
Nesting is recursive.

```text
a[
    b[c -> d]
    b -> e
]
```

Identifiers continue to refer to the same box throughout the diagram, including
inside nested expressions. Leaf boxes have equal dimensions. Container boxes
expand to fit their contents.

## Layout

Connections flow from left to right. There are no layout-direction directives.
When a diagram does not fit horizontally, the renderer introduces additional
horizontal levels and routes connections between them.

## Provisional grammar

```text
diagram   = statement { newline statement }
statement = box { "->" box }
box       = identifier [ "[" (quoted-text | diagram) "]" ]
```

## Not yet specified

- Box and connection styling.
- Padding, margins, and spacing defaults.
- Whether a container can have both text and nested contents.
- How conflicting text declarations for the same identifier are handled.
- Whether the same box may belong to more than one container.
