---
name: memory-hygiene
description: "Keeps the knowledge bank trustworthy: finds stale, low-confidence, duplicated and conflicting items with knowledge_review and resolves each with a verdict. Load when a pack shows outdated or contradictory knowledge, when the bank has grown large, or as a periodic tidy-up after a project milestone."
---

# Memory Hygiene

## Purpose

A memory that is never cleaned becomes a source of confident mistakes. This sub-skill finds what has gone stale or inconsistent and resolves it. Hands back the verdicts applied and what was left for a person to decide.

## What counts as a problem

| Problem | Test | Policy key |
| --- | --- | --- |
| stale | not confirmed for longer than the stale window | `hygiene.stale_after_days` |
| weak | effective confidence below the prune line | `hygiene.prune_below` |
| duplicate | two active items whose texts are at least this similar | `hygiene.duplicate_similarity` |
| conflict | same subject, similar enough to be about one thing, different enough to disagree | `hygiene.conflict_similarity` up to the duplicate line |

Similarity is word overlap after dropping stopwords: crude, predictable and the same on the server and by hand.

## Procedure

1. Call `knowledge_review` for the workspace. It returns up to `hygiene.review_batch` of each problem.
2. **Duplicates:** keep the better-sourced one; supersede the other into it or retire it.
3. **Conflicts:** check the sources. If one source is newer or more authoritative, supersede the other. If the sources themselves disagree, do not pick a winner silently: leave a comment for the person on the newer source and retire neither.
4. **Stale:** confirm what the current pages still support, contradict what they no longer do, retire what no longer applies. Do not confirm from memory; confirm from a source read in this session.
5. **Weak:** retire, unless the item is a decision whose source still exists; then confirm or supersede from that source.

## Gate

Every item returned by the review has a verdict or an open question addressed to a person, and no verdict was given without reading the source it rests on.

## Hands back

The verdicts and the open questions, to the agent or person who asked for the review.
