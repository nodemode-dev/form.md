---
name: handoff
description: "Writes and reads handoffs between agents and sessions: goal, state, what is done, what is open, decisions, sources and the next step, kept within a fixed size, and paired with a delta pack for the receiver. Load when stopping mid-task, passing work to another agent or tool, or resuming work someone else started."
---

# Handoff

## Purpose

Let the next agent continue without re-reading everything or asking what happened. Hands back a handoff note and the pack id the receiver should ask a delta from.

## The note

Fields, in this order (`handoff.fields`): goal, state, done, open, decisions, sources, next. At most `handoff.max_open_items` open items and `handoff.max_chars` characters in all; past that, the work was not broken down enough to hand off and should be split.

- **goal** in the person's words, one sentence.
- **state** one line: working, blocked (on what), or waiting (for whom).
- **done** what changed, each with where (page and block ids, or files and lines).
- **open** what remains, most important first, each small enough for one session.
- **decisions** each one also recorded in the knowledge bank; here only its id and one line.
- **sources** the pack id and the few sources that mattered most.
- **next** the single next action, concrete enough to start without a question.

## Procedure

### Giving

1. Record every decision with knowledge_record first, so the note can point at ids instead of restating them.
2. Write the note. Where the work lives in form.md, put it in the page or comment thread the work belongs to; in a repository, in the place the project uses for notes.
3. Take a `context_pack` with `task_type: "handoff"` and keep its `pack_id` in the note's sources.

### Receiving

1. Read the note, then call `context_pack` with `format: "delta"` and `since` set to the giver's pack id. The delta shows what changed after the note was written, which the note cannot.
2. Confirm or contradict the decisions you rely on, as in the knowledge bank.
3. If `next` is unclear or the delta contradicts the note, ask the person one question before acting.

## Gate

The note fits its limits, every decision in it has a knowledge id, and `next` can be started without a question.

## Hands back

The note and the pack id.
