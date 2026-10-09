---
name: compression-formats
description: "Shapes packed material as a digest (the leading sentences of each passage), an outline (the heading tree of each page) or a delta (only what changed since a time or an earlier pack). Load when choosing how to present context, when a pack is too long for its budget, or when resuming work."
---

# Compression Formats

## Purpose

Fit more meaning into fewer tokens without losing what the task depends on. Hands back each item in the chosen format, with its source kept.

## The formats

| Format | Keeps | Limits in `policy.json` | Use for |
| --- | --- | --- | --- |
| digest | the heading path and the first sentences of each passage | `compression.digest.sentences_per_passage`, `max_passage_chars` | answering and editing, the default (`compression.default_format`) |
| outline | each page's heading tree, no body text | `compression.outline.max_depth`, `max_headings_per_page` | planning, reviewing structure, deciding where a change goes |
| delta | only pages and knowledge changed since a time or a previous pack | `compression.delta.default_hours`, `max_hours` | resuming, handoffs, checking what a collaborator did |

## Procedure

1. **Pick by task, not by habit.** Planning wants the outline; editing a paragraph wants the digest of that area plus the outline around it; resuming wants the delta.
2. **Never compress the rules or the exact text you will change.** Compression is for orientation. The block or lines you will edit are read in full, by id or line range, right before editing.
3. **Keep identifiers verbatim.** Names, ids, numbers, dates and quoted phrases survive compression untouched; it is prose around them that is cut.
4. **Mark what was cut.** A digest ends a passage with an ellipsis; an outline says how many headings it omitted. An agent that cannot tell a summary from the whole will act on the summary.
5. **Delta needs an anchor.** Pass `since` as an earlier pack id or a time. Without one the default window applies; if that window is longer than `max_hours`, take a fresh pack instead.

## Gate

Every compressed item names its source, identifiers are unchanged, and nothing the task will modify was only seen compressed.

## Hands back

The formatted items to context-packing or handoff.
