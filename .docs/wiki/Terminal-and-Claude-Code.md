# Terminal and Claude Code

Since v0.14.0 form.md has a bottom panel with terminals, Claude Code and a Claude Code chat, the same harness formvis and formlit ship.

## Who can use it

The instance owner, signed in, on a form.md that listens on this computer (`127.0.0.1`, the default for `./form-md` and the desktop case). A terminal is a shell as the user the server runs as, so on a server it is off unless you set `FORMMD_TERMINAL=on`; `FORMMD_TERMINAL=off` turns it off even locally. Agent tokens never reach it. A terminal tab is attached through the browser session that opened it: signing that session out (here, from another device in **Account settings, Sessions**, or by an admin) detaches it at once, while the shell keeps running for your other sessions. Windows servers report that the terminal needs macOS or Linux.

## The panel

| Keys | Opens |
|---|---|
| `Mod+J` | the panel (or the terminal button in the top bar) |
| `Mod+Alt+J` | Claude Code in a new tab |
| `Mod+Alt+K` | the Claude Code chat |

Terminal tabs keep running when the panel is hidden or the page reloads; reopening replays the last 256 KiB. Double-click a tab to restart its process, the x closes it. Drag the top edge to resize the panel.

## Claude Code

Claude Code must be installed and signed in once (`claude` in any terminal). form.md finds it on the PATH, in the usual install folders, through your login shell, or at `FORMMD_CLAUDE_BIN`. Each tab starts in the data directory's `agent/` folder with this form.md's MCP endpoint attached through a token named "Claude Code in form.md" in your account's API tokens; revoking it there cuts the in-app agent off until the next server start.

The chat runs the same Claude Code non-interactively and shows its answer, every form.md tool it calls and the memory it uses. **Open in Claude Code** continues the conversation in a terminal tab; **From Claude Code** brings a tab's conversation into the chat.

Outside the app, `./form-md claude` starts Claude Code in your own terminal with form.md's tools, and `./form-md agents install` registers form.md with Claude Code for every project; both need `FORMMD_MCP_TOKEN` set to a personal API token.

## Driving the window

Every command in the window has an id, a label, a group and, for newer commands, a description, typed parameters and whether it changes anything. Agents use:

| Tool or route | Does |
|---|---|
| MCP `window_commands` · `GET /api/commands` | lists the open window's commands |
| MCP `window_command_describe` · `GET /api/commands/{id}` | one command in full |
| MCP `window_command_run` · `POST /api/commands/run` | runs it in the window and returns its result |
| `./form-md cli list`, `describe ID`, `run ID key=value` | the same from a terminal, with `FORMMD_MCP_TOKEN` |

A call needs a form.md window open for the same account; otherwise it fails with a message saying so. A read-only token can run only commands that change nothing.
