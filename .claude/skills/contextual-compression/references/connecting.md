# Connecting agents

How Claude Code, Codex, Cursor and other MCP clients reach a form.md workspace, which tools this skill uses, and how to tell a connection fault from a skill fault.

## Contents

- The tools
- Prompts and resources
- Claude Code
- Codex
- Cursor
- Other clients and the stdio bridge
- Extensions in the window
- When a call fails

## The tools

| Tool | Writes | Use |
| --- | --- | --- |
| `context_pack` | no | a ranked, budgeted pack for a task; arguments `task`, `task_type`, `workspace_id`, `focus`, `format`, `since`, `budget`, `agent_key` |
| `knowledge_search` | no | find knowledge items; `query`, `workspace_id`, `kind`, `limit` |
| `knowledge_review` | no | stale, weak, duplicate and conflicting items; `workspace_id` |
| `knowledge_record` | yes | record an item; `workspace_id`, `kind`, `subject`, `text`, `source_page`, `source_block` |
| `knowledge_confirm` | yes | `id` and `verdict` (confirm, contradict, supersede with `text`, retire) |
| `family_list` | no | the other form family apps on this computer (formlit, formvis): purpose, running or not, MCP endpoint, tool count, how to connect |
| `family_route` | no | rank form.md, formlit and formvis for a `task` and name the sibling tools that match; see `sub-skills/family-routing/SUB-SKILL.md` |

They sit beside the form-md skill's page and database tools on the same server. A read-only credential sees only the first three.

## Prompts and resources

The server offers this skill as MCP prompts (one per sub-skill, named after it, plus `contextual-compression` for the router) and as resources under `skill://contextual-compression/`, including `policy.json`. A client that supports prompts can load the method without having the files installed.

## Claude Code

Add the server, then install the skill into the project or your user skills folder:

```bash
claude mcp add --transport http form-md https://[your-host]/mcp --header "Authorization: Bearer [token]"
cp -r [form.md checkout]/.claude/skills/contextual-compression .claude/skills/
```

Or commit a `.mcp.json` at the project root:

```json
{ "mcpServers": { "form-md": { "type": "http", "url": "https://[your-host]/mcp", "headers": { "Authorization": "Bearer ${FORMMD_MCP_TOKEN}" } } } }
```

Without a token header, Claude Code signs in through OAuth and form.md shows its consent page.

## Codex

Codex runs MCP servers as local commands, so use the stdio bridge built into the form-md binary. In `~/.codex/config.toml`:

```toml
[mcp_servers.form-md]
command = "form-md"
args = ["mcp"]
env = { FORMMD_MCP_URL = "https://[your-host]", FORMMD_MCP_TOKEN = "[token]" }
```

Point Codex at the method from the project's `AGENTS.md`: name this skill's folder and the loop in its SKILL.md.

## Cursor

In `.cursor/mcp.json` at the project root, or the global one:

```json
{ "mcpServers": { "form-md": { "url": "https://[your-host]/mcp", "headers": { "Authorization": "Bearer [token]" } } } }
```

Add a project rule in `.cursor/rules/` that tells the agent to take a `context_pack` before a task and to follow this skill's SKILL.md.

## Other clients and the stdio bridge

`form-md mcp` reads JSON-RPC messages from standard input, one per line, forwards each to `[FORMMD_MCP_URL]/mcp` with the token, and writes each answer to standard output. Any client that can start a command can use it. `--url` and `--token` override the environment.

## Extensions in the window

Form extensions installed with FormPortal or run from form.ext live in the person's open window. `extensions_list` shows them (id, version, source, state, commands) and `extension_command` runs one of their commands with `id`, `command` and `args`. Both go through the window like `window_command_run`, so they fail when no window is open, and the window asks the person to allow an extension version the first time; a refusal is the person's answer, not a fault to retry. From a terminal: `form-md cli extensions` and `form-md cli extension ID COMMAND`.

## When a call fails

1. **Unknown tool.** The server is older than this skill, or the credential is read-only and the tool writes. Ask `tools/list` what exists.
2. **401.** The token is wrong or revoked, or the OAuth sign-in expired; sign in again.
3. **Not found for a workspace.** The credential does not reach it, or the workspace closed itself to agents. `whoami` lists what it reaches.
4. **The pack is empty or off-target.** That is a skill question, not a connection fault: check the task words and the focus, then see relevance-ranking.

To separate a server fault from a skill fault, call the tool directly with minimal arguments and no skill loaded. If that fails, no change to the skill will fix it.
