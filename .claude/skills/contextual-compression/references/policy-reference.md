# policy.json reference

`policy.json` is the single source of every number in this skill. The form.md server embeds the same file at build time (`internal/contextpack` reads it), the MCP prompts and resources serve it, and `make check` refuses a file that does not parse or breaks the rules below. Change a number here and both the agents following this skill and the server packing context change together.

## Contents

- tokens
- budgets
- sections
- ranking
- knowledge
- compression
- handoff
- hygiene
- Tuning

## tokens

`chars_per_token`: the estimate used to turn characters into tokens. Four is close for English prose and code; set it lower for languages with long words per token.

## budgets

- `tasks`: the budget in tokens for each task type: answer, edit, build, review, plan, handoff.
- `default_task`: the type used when none is given.
- `min`, `max`: every explicit budget is clamped to this range.

## sections

An ordered list. Each entry has a `kind` (rules, knowledge, focus, passages, links), a `share` of the budget, and `required`. Shares should add up to 1. A section that does not use its share passes the rest down the list. Required sections are filled even if that trims the later ones.

## ranking

- `weights`: text, title, link, recency, knowledge, focus. Relative, not normalized; raising one is the same as lowering all others.
- `recency_half_life_days`: the recency signal halves every this many days since a page changed.
- `max_candidates`: how many candidates are scored at all.
- `min_score`: candidates below this are dropped.
- `per_page`: most passages kept from one page.
- `stopwords`: words ignored when matching a task's text.

## knowledge

- `kinds`: the allowed kinds. Adding a kind needs a server change too, because the database checks it.
- `max_text_chars`, `max_subject_chars`: longer input is refused, not cut.
- `max_items_per_workspace`: recording past this asks for a hygiene pass first.
- `confidence`: `initial` for sourced items, `agent_initial` for items with no source, `confirm_step` and `contradict_step` per verdict, `max` cap, `floor` below which an item is never shown.
- `half_life_days`: per kind; effective confidence halves every this many days since the last confirmation.
- `pack_limit`: most knowledge items in one pack. `search_limit`: default for knowledge_search.

## compression

- `formats` and `default_format`.
- `digest`: `sentences_per_passage`, `max_passage_chars`.
- `outline`: `max_depth` (heading levels), `max_headings_per_page`.
- `delta`: `default_hours` when no `since` is given, `max_hours` beyond which a fresh pack is better.

## handoff

`fields` in order, `max_open_items`, `max_chars`.

## hygiene

- `duplicate_similarity`: word-overlap ratio at or above which two items are the same.
- `conflict_similarity`: same subject and at least this similar, but below the duplicate line, means they may disagree.
- `stale_after_days`, `prune_below`, `review_batch`.

## Tuning

Change one number at a time and say why in the commit. Shares outside 0 to 1, weights below 0, a `min` above `max`, a default task or format that is not listed, or a duplicate line below the conflict line are refused by the check. After changing a budget, take a pack for a real task before and after and compare what was dropped.
