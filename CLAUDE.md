# form.md

A self-hosted workspace for people and agents in one Go binary: documents, databases and realtime editing, with an MCP server for agents. Backend: Go 1.26, pure-Go SQLite, a Yjs CRDT in Go. Frontend: vanilla TypeScript, Vite, ProseMirror and Yjs, no UI framework.

## Repo layout notes

- `assets/` holds repo-level art such as the logo. Not app runtime assets; the client's assets live under `web/`.
- `.docs/wiki/` is the editable source of the GitHub wiki; `go run ./tools/wikicheck` keeps it in step with the code.
- `.claude/skills/contextual-compression/` is a project skill and also part of the build: `embed.go` embeds it, `internal/contextpack` reads its `policy.json`, and the MCP server serves it as prompts and resources. Changing a number there changes the server.
- `examples/agents/` holds client configs to copy (Claude Code, Codex, Cursor). `.mcp.json` and `.cursor/mcp.json` connect agents working on this repo to a local form.md at `http://127.0.0.1:8750` with `FORMMD_MCP_TOKEN`.
- `form-md`, `form-md.bat` and the `*_oc_install` scripts at the root are the boot and install scripts the repo ships, not build output.

## Skills

- `contextual-compression`: how an agent gets the smallest relevant context for a task, keeps a sourced knowledge bank, hands work off and keeps memory clean. Use it at the start of any task on a form.md workspace (the `context_pack` tool) and, by hand, on this repo. Route through its `SKILL.md`; the sub-skills cover packing, ranking, formats, the knowledge bank, handoff and hygiene.

## Checks

`make check` runs gofmt, go vet, the wiki check, the skill check (policy.json must parse and pass its rules) and the TypeScript check.

Always start work here by routing through the `contextual-compression` skill: pack the context the task needs before reading the tree file by file.
