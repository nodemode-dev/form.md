---
name: context-packing
description: "Builds a context pack for a task: picks the budget from the task type, fills the rules, knowledge, focus, passages and links sections in that order of priority, and stops at the budget. Load at the start of any task, and whenever the current context no longer fits the work."
---

# Context Packing

## Purpose

Turn a task into a pack: the smallest body of context that lets the agent do the task without guessing. Hands back the pack (text plus its list of sources) and the budget it used.

## Inputs

The task in the person's own words, its type, any page or file the person pointed at, and the workspace or repository. The task type picks the budget:

| Task type | Use when | Budget key |
| --- | --- | --- |
| answer | a question that needs facts, no change | `budgets.tasks.answer` |
| edit | changing one page, file or small area | `budgets.tasks.edit` |
| build | new feature, new database, several files | `budgets.tasks.build` |
| review | checking work against rules and intent | `budgets.tasks.review` |
| plan | deciding what to do before doing it | `budgets.tasks.plan` |
| handoff | resuming or passing on work | `budgets.tasks.handoff` |

When unsure, use `budgets.default_task`. A budget given explicitly is clamped to `budgets.min` and `budgets.max`. Tokens are estimated as characters divided by `tokens.chars_per_token`, which is deliberately crude: the pack aims for the right material, not an exact count.

## Procedure

### 1. Connected to form.md

Call `context_pack` with `task`, `task_type`, `workspace_id`, `focus` (page ids the person named or is looking at) and, when resuming, `format: "delta"` with `since`. Read the pack top to bottom before calling anything else. Its sources list says why each item is there (`why`: text match, title, link to focus, recent, knowledge), which tells you what is safe to skip.

### 2. In a plain repository

Do the same by hand, in this order, stopping when the budget is spent:

1. **Rules.** The project's agent instructions (`CLAUDE.md`, `AGENTS.md`, contributor rules), always first, never trimmed. Share: `sections` entry `rules`.
2. **Knowledge.** Decisions and facts that match the task: the knowledge bank if one exists, otherwise decision records, build notes and change logs. Share: `knowledge`.
3. **Focus.** The files or pages the person named, in outline first; full text only for the part you will change. Share: `focus`.
4. **Passages.** Search hits for the task's words, ranked as in relevance-ranking, at most `ranking.per_page` per file. Share: `passages`.
5. **Links.** Titles and one line each for what the focus links to or is linked from. Share: `links`.

A section that does not use its share passes the rest down the list. Required sections (`required: true`) are filled even when that means trimming the others.

### 3. Check the pack

Before starting work, confirm the pack answers three questions: what the person wants, what constraints apply, and where the change goes. If one is unanswered, make one targeted read or ask one question; do not widen the pack wholesale.

## Gate

The pack fits its budget, starts with the rules, names a source for every item, and answers the three questions above.

## Hands back

The pack, the sources with their reasons, and the budget used, to the router or to the agent doing the work. For ordering decisions load `sub-skills/relevance-ranking/SUB-SKILL.md`; for the shape of each item load `sub-skills/compression-formats/SUB-SKILL.md`.
