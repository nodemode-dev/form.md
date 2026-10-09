---
name: relevance-ranking
description: "Scores candidate passages, pages and knowledge for a task by text match, title match, links to the focus, recency and recorded knowledge, with weights from policy.json. Load when there is more material than budget, when a pack feels off-target, or when explaining why something was included."
---

# Relevance Ranking

## Purpose

Order candidates so the budget is spent on what the task needs. Hands back a ranked list where every entry carries its score and the signals that produced it.

## The signals

Each candidate gets a score in roughly zero to one per signal, multiplied by its weight in `ranking.weights` and summed.

| Signal | Weight key | What it measures |
| --- | --- | --- |
| text | `text` | how well the passage matches the task's words, after dropping `ranking.stopwords`; full-text rank when the server does it |
| title | `title` | share of the task's words that appear in the page title |
| link | `link` | the page links to, or is linked from, a focus page |
| recency | `recency` | halves every `ranking.recency_half_life_days` since the page last changed |
| knowledge | `knowledge` | a knowledge item that matches the task names this page as its source |
| focus | `focus` | the candidate is a focus page or sits inside one |

Candidates below `ranking.min_score` are dropped. At most `ranking.per_page` passages are kept from any one page, so one long page cannot crowd out the rest. At most `ranking.max_candidates` are scored at all.

## Procedure

1. **Extract the task's words.** Lowercase, drop stopwords, keep names, identifiers and quoted phrases intact. An identifier such as a function or property name is worth more than a common word; a person's own phrase is worth more than a paraphrase.
2. **Gather candidates broadly.** Match any of the words, not all of them: a task description is a sentence, and requiring every word returns nothing.
3. **Score and cut.** Apply the weights, drop below the floor, cap per page.
4. **Read the top of the list critically.** If the top three are all recent but off-topic, recency is over-weighted for this task; if they are all one page, raise diversity by lowering `per_page` for this pack. Change the policy only when the same correction keeps recurring, and then through `references/policy-reference.md`.

## Gate

Every kept candidate has a score above the floor and at least one named signal, and no page holds more than its per-page share.

## Hands back

The ranked candidates with scores and reasons, to context-packing.
