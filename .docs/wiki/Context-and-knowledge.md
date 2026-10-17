# Context and knowledge

Agents do their best work when they start a task with exactly the context it needs. form.md packs that context for them: the workspace's rules, what people and agents have recorded as knowledge, the pages the task is about, the best matching passages and the pages around them, ranked and cut to a token budget. The method lives in the `contextual-compression` skill in `.claude/skills/contextual-compression/`, and the server reads the same skill: its `policy.json` holds every budget, weight and threshold, so the skill and the code never disagree.

## Layout

| Path | What lives there |
|---|---|
| `.claude/skills/contextual-compression/` | The skill: router `SKILL.md`, seven sub-skills (`family-routing` among them), `policy.json`, references and evaluations |
| `embed.go` | Embeds the skill into the binary (`ContextSkill`) |
| `internal/contextpack/` | Policy loading and validation, the knowledge bank, packing, ranking, compression, hygiene review, undo |
| `internal/search/context.go` | Passages matching any word of a task, with rank and heading |
| `internal/mcp/tools_context.go` | The `context_pack` and `knowledge_*` tools, the skill as MCP prompts and resources |
| `cmd/form-md/bridge.go` | `form-md mcp`, the stdio bridge for clients that start a command |
| `tools/skillcheck/` | `make check` step: `policy.json` parses and passes its rules, frontmatter and routing are right |
| `examples/agents/` | Configs to copy for Claude Code, Codex and Cursor |

## The tools

| Tool | Writes | What it does |
|---|---|---|
| `context_pack` | no | A pack for a task: `task`, `task_type` (answer, edit, build, review, plan, handoff), `focus` page ids, `format` (digest, outline or delta), `since`, `budget`, `agent_key` |
| `knowledge_search` | no | Knowledge items matching words, most confident first, optionally of one kind |
| `knowledge_review` | no | Stale, weak, likely duplicate and possibly conflicting items |
| `knowledge_record` | yes | One fact, decision, glossary term, constraint or preference, with the page and block it comes from |
| `knowledge_confirm` | yes | A verdict on an item: confirm, contradict, supersede with corrected text, or retire |

A read-only credential sees the first three. Knowledge writes are part of the agent run, so **Undo this run** removes recorded items and puts verdicts back.

The server also offers the skill as MCP prompts (`contextual-compression` and one per sub-skill) and as resources under `skill://contextual-compression/`, including `policy.json`. The skill download in Account settings, Connected apps carries the folder too.

## How a pack is built

1. The budget is the task type's entry in `budgets.tasks`, or the explicit `budget` clamped between `budgets.min` and `budgets.max`.
2. Sections run in the order of `sections`, each with its share of the budget plus whatever earlier sections left. Rules and knowledge are required; rules are never cut.
3. Knowledge is ranked by word match times effective confidence, which halves every `knowledge.half_life_days[kind]` since the item was last confirmed.
4. A focus page goes in whole when it fits, otherwise as a digest of each section, otherwise as an outline.
5. Passages are scored by text match, title match, links to a focus page, recency, recorded knowledge naming the page, and sitting inside a focus page, weighted by `ranking.weights`. Below `ranking.min_score` they are dropped; one page keeps at most `ranking.per_page`.
6. Related pages list what links to or from a focus page, the sources of the knowledge shown, and in a delta every page changed since the cutoff.
7. When formlit or formvis fits the task better than form.md (at least `family.pack_min_score`), a short Form family note names it, says whether it is running and how to connect, and lists up to `family.pack_max_tools` of its matching tools. It sits outside the section budget and is counted in the pack's total. See [[Form-family]].

Every pack has an id. Pass it later as `since` with `format` delta and the next pack holds only what changed. A pack only ever contains what the credential may read, and knowledge sourced from a page someone cannot open is hidden from them.

The workspace agent and custom agents get the same knowledge in their instructions and the same tools.

## Connecting clients

**Claude Code:** `claude mcp add --transport http form-md https://notes.example.com/mcp`, or a project `.mcp.json` like `examples/agents/claude-code.mcp.json`. Copy the skill folder into `.claude/skills/` to have the method without the server.

**Codex:** Codex starts MCP servers as commands, so it runs the bridge. In `~/.codex/config.toml`:

```toml
[mcp_servers.form-md]
command = "form-md"
args = ["mcp"]
env = { FORMMD_MCP_URL = "https://notes.example.com", FORMMD_MCP_TOKEN = "fmd_..." }
```

**Cursor:** `.cursor/mcp.json` like `examples/agents/cursor.mcp.json`, plus a rule in `.cursor/rules/` that tells the agent to take a pack first (this repo ships one).

**Anything that starts a command:** `form-md mcp` reads one JSON-RPC message per line on standard input, forwards it to `FORMMD_MCP_URL` (default `http://127.0.0.1:8750`) with `FORMMD_MCP_TOKEN`, and writes each answer as one line. `--url` and `--token` override the variables. A refused token or an unreachable server comes back as a JSON-RPC error.

## Changing the policy

Edit `policy.json`, run `make check`, rebuild. The skill's `references/policy-reference.md` says what each key does. Shares must add up to 1, weights must not be negative, the duplicate line must sit above the conflict line, and `knowledge.kinds` must match the database's list; the server refuses to start with a policy that breaks a rule and says which.

## Related

[[API-and-agents]] · [[Form-family]] · [[Development]]
