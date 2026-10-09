---
name: knowledge-bank
description: "Records, finds, confirms, contradicts, supersedes and retires durable knowledge: facts, decisions, glossary terms, constraints and preferences, each with a subject, a source page and block, a confidence and a last-confirmed date. Load when something learned or decided will matter to a later task, when a pack's knowledge proved right or wrong, or when a term needs a fixed meaning."
---

# Knowledge Bank

## Purpose

Keep a small, sourced, self-correcting memory per workspace (or per repository) so that later tasks start with what earlier ones established. Hands back the recorded or updated items with their ids.

## What belongs in the bank

| Kind | Example | Half-life key |
| --- | --- | --- |
| fact | "Invoices are numbered per workspace, starting at 1000." | `knowledge.half_life_days.fact` |
| decision | "Exports use the Go fonts; no font files ship." with why | `knowledge.half_life_days.decision` |
| glossary | subject "run", text "a group of agent edits that undo together" | `knowledge.half_life_days.glossary` |
| constraint | "The up-front script stays under 250 KB gzipped." | `knowledge.half_life_days.constraint` |
| preference | "Ada wants German summaries for the board." | `knowledge.half_life_days.preference` |

What does not belong: anything already stated plainly in a page the pack will find anyway, temporary state of the current task (that is a handoff), secrets, and personal data the person did not ask to keep.

## Procedure

### Record

Call `knowledge_record` with `kind`, a short `subject` (at most `knowledge.max_subject_chars`), the `text` in one or two plain sentences (at most `knowledge.max_text_chars`), and `source_page` plus `source_block` whenever the knowledge comes from somewhere. Write it so it stands alone: a reader with no context must understand it.

The server compares the new text with what is already there. A near-identical item (similarity at or above `hygiene.duplicate_similarity`) is confirmed instead of duplicated, and the answer says so with `merged: true`. Items on the same subject that differ are returned as `conflicts`; resolve them with knowledge_confirm rather than leaving two truths.

### Find

Call `knowledge_search` with the task's words. Results are ordered by match times effective confidence. Effective confidence is the stored confidence halved every half-life of its kind since it was last confirmed, so an old, never-confirmed fact ranks below a fresh one.

### Confirm, contradict, supersede, retire

Call `knowledge_confirm` with the item `id` and a `verdict`:

| Verdict | When | Effect |
| --- | --- | --- |
| confirm | the work relied on it and it held | confidence rises by `confidence.confirm_step` up to `confidence.max`; last-confirmed resets |
| contradict | the work showed it is wrong or doubtful | confidence falls by `confidence.contradict_step` |
| supersede | it changed; pass the new `text` | the old item is retired and linked to a new one |
| retire | it no longer applies | it leaves every pack and search |

### Without a server

In a plain repository keep the same fields as one JSON object per line in a file the project agrees on, and apply the same verdicts by editing that line. The rules above do not change.

## Gate

Every recorded item stands alone, carries a kind and subject, names its source or is explicitly sourceless, and no two active items on one subject disagree without a pending verdict.

## Hands back

Item ids, merges and conflicts, to the agent doing the work; conflicts that need judgment go to `sub-skills/memory-hygiene/SUB-SKILL.md`.
