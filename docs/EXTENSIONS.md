# Extensions in form.md

form.md hosts extensions written to the Form Extension Contract, API 1. An extension is a folder (or a `.formpkg` that FormPortal unpacks) with a `manifest.json` whose `product` is `formmd`. It runs in a sandboxed frame inside the form.md window and talks to it only through the contract's JSON-RPC messages. This page is the form.md part of the contract: what `document.read` returns, which commands `document.apply` and `app.run` take, which events form.md sends, and where extensions appear.

## The document

The document is the page open in the editor. Pages are made of blocks; every block has a stable `id` that survives edits by other people, so extensions address blocks by id, never by position. A database page has no editor document; `document.read` then returns the page with an empty block list.

### `document.read`

`params.what` picks a part. Every answer carries `page`, `writable` and `mode`; the rest depends on `what`.

| `what` | Adds |
|---|---|
| `all` (default) | `blocks`, `markdown`, `selection`, `outline` |
| `blocks` | `blocks` |
| `markdown` | `markdown` |
| `selection` | `selection` |
| `outline` | `outline` (the headings) |
| `page` | nothing more |
| `file` | refused with -32004: a form.md page is not a file, so read it as Markdown instead |

Any other `what` answers -32602.

A real snapshot of a short untitled page, `what: "all"`, taken from the running app by a test extension:

```json
{
  "page": { "id": "pb434ysqsdq2qismuikddykytu", "title": "", "kind": "doc", "icon": "", "workspace_id": "ia2673mz4jc4nrcibivg4yicpi", "url": "/p/pb434ysqsdq2qismuikddykytu" },
  "writable": true,
  "mode": "edit",
  "blocks": [
    { "id": "3gi3mfzeze", "type": "heading", "text": "Goals", "level": 1 },
    { "id": "ct7yjwapiu", "type": "paragraph", "text": "Ship the beta to the first ten teams." }
  ],
  "markdown": "# Goals\n\nShip the beta to the first ten teams.\n",
  "selection": { "block": "ct7yjwapiu", "from": 37, "to": 37, "empty": true, "text": "" },
  "outline": [{ "id": "3gi3mfzeze", "level": 1, "text": "Goals" }]
}
```

- `page` is `null` when no page is open (Home, Library, Trash, Inbox); `route` then names the view.
- `blocks` are the top-level blocks in order. `type` is the block type (`paragraph`, `heading`, `blockquote`, `callout`, `bullet_list`, `ordered_list`, `task_list`, `toggle`, `code_block`, `table`, `image`, `file`, `bookmark`, `embed`, `math_block`, `columns`, `database`, `synced`, `toc`, `horizontal_rule` and others); `level` is set for headings, `language` for code blocks, `checked` lists the ticks of a to-do list. `text` is the block's plain text, at most 4000 characters, with list items on their own lines.
- `markdown` is the whole page as form.md writes Markdown (the same as Copy page as Markdown).
- `selection` is the person's last selection in the page: the block it starts in, offsets inside that block's text, and the selected text (at most 2000 characters).
- `mode` is `edit`, `suggest` or `view`; `writable` is true only in `edit` with edit rights. Changes are refused when `writable` is false.

### `document.apply`

`{ label, ops: [{ command, args }] }` runs every op in order as one change: one undo step (Undo, Mod+Z, undoes all of it), one sync to everyone on the page. The ops are form.md's own `doc.*` commands, the same ones its command palette, `form-md cli` and MCP `window_command_run` use. If one op fails, nothing is applied and the error names the op. The result is `{ applied, results }` with one result per op.

| Command | Args | Result |
|---|---|---|
| `doc.insertMarkdown` | `markdown`, `at`: `end` (default), `start`, `cursor`, or `after` with `block` | `{ blocks: [ids] }` |
| `doc.insertText` | `text`, `at`, `block`. With `at: cursor` and text with no blank line, the text goes inline at the selection in its text block (replacing selected text, no marks); otherwise it becomes paragraphs (split on blank lines) | `{ blocks: [ids] }`, or `{ blocks: [], inline: id }` for inline text |
| `doc.replaceBlock` | `block`, `markdown` | `{ blocks: [ids] }` |
| `doc.setBlockText` | `block`, `text` (keeps the block's type; text blocks only) | `{ block }` |
| `doc.deleteBlock` | `block` | `{ deleted: id }` |
| `doc.moveBlock` | `block`, `index` (0 is first) | `{ block, index }` |
| `doc.replaceText` | `find`, `replace`, `all` (default true), `caseSensitive` (default false) | `{ replaced: n }` |
| `doc.markText` | `find`, `mark` (`strong`, `em`, `underline`, `strike`, `code`, `highlight`), `all` | `{ marked: n }` |
| `doc.select` | `block`, optional `from`, `to` (offsets in the block's text) | `{ block }` |

Two more `doc.*` commands go through `app.run`, since they are not edits: `doc.undo` and `doc.redo` (the person's own undo stack on the open page, with no need for the editor to have focus).

### `app.commands` and `app.run`

`app.commands` returns the window's catalogue as `[{ id, title, group, args?, mutates, description? }]`, the same list as Find a Command (Mod+Shift+P). `app.run` runs one of them with `args` and returns its result. Useful ones for extensions:

| Command | Args | Does |
|---|---|---|
| `doc.undo`, `doc.redo` | none | undo or redo on the open page |
| `page.new` | none | a new page, opened |
| `page.newInside` | none | a sub-page of the open page |
| `page.newDatabase` | none | a new database page |
| `page.copyMarkdown` | none | copies the page as Markdown |
| `page.favorite` | none | adds or removes the open page from Favorites |
| `page.find` | none | opens Find and Replace |
| `page.comments` | none | shows the comments pane |
| `search.open` | none | opens Search |
| `route.home`, `library.open`, `trash.open` | none | goes to Home, Library or Trash |
| `nav.back`, `nav.forward` | none | history |
| `view.focus`, `view.typewriter`, `view.wordCount` | none | writing views |
| `shell.theme.toggle` | none | dark or light |
| `templates.open` | none | the template gallery |

Commands marked risky in the catalogue, and the `extensions.*` commands themselves, are refused with -32004. Editing commands that work on the cursor (`editor.bold`, `editor.h1` and the like) need the editor to have focus, which it does not while an extension panel does; use the `doc.*` commands instead.

## Events (`ext.event`)

| Name | Data | When |
|---|---|---|
| `document.changed` | `{ page, reason: "edit" \| "opened", blocks }` | the open page's content changed (by anyone, 250 ms after the last change), or another page was opened (`blocks` is the number of top-level blocks) |
| `selection.changed` | `{ page, block, from, to, empty, text }` | the person's selection in the page moved (150 ms after it settles) |

## Files and network

- `files.open` (`files-read`): the browser's own file picker, filtered by `accept`; at most 200 MB in total.
- `files.save` (`files-write`): form.md has no save dialog of its own, so it asks the person to confirm the name in its own dialog and the browser saves the file as a download into the downloads folder (the browser adds a number when the name is taken). Cancel answers -32002.
- `net.fetch` is not available in form.md: it answers -32003 to an extension that holds the `network` permission (and -32001, as for every method, to one that does not): the window reaches only its own server.
- `storage.get` and `storage.set` keep up to 256 KB per extension in this browser.

## Where extensions appear

- **Extensions menu**: the Extensions item in the sidebar foot, above Settings, lists one item per contributed command (`Name: Title`), then Extensions Panel, Reload Extensions and Manage in FormPortal. In the desktop app the menu bar also has an Extensions menu with Extension Commands (opens the same list), Extensions Panel, Reload Extensions and Manage in FormPortal.
- **Find a Command** (Mod+Shift+P) lists every contributed command under Extensions.
- **Extensions panel**: a pane on the right, like the agent and comments panes. One row per extension with name, version, source, permission chips and state; Open for each of its panels; the folder for development extensions.
- **Contributed panels**: a `dock` panel opens as a tab of that right-side pane, titled with the extension's name; a `dialog` panel opens as a form.md dialog.

## Agents

- MCP `extensions_list`: every extension with id, name, version, source, state (`ready`, `needs-permission`, `refused`), reason, permissions, panels and commands (in the result's structured content).
- MCP `extension_command` with `id`, `command`, `args`: runs a contributed command in the open window. The window asks for consent first if the person has not allowed that version yet; the call waits up to 60 s.
- `form-md cli extensions` and `form-md cli extension ID COMMAND [key=value...]` do the same from a terminal.

## Running an extension

- **From a folder (form.ext Run in App)**: form.ext writes `extensions/formmd/dev.json` under the FormPortal root with the folder's path. form.md checks the file every 2 s while its window has focus, and on Reload Extensions, and shows the extension with the source Development and its folder.
- **From FormPortal**: FormPortal unpacks the package and writes `extensions/formmd/enabled.json`; form.md picks it up the same way, with the source Catalogue or File.
- The FormPortal root is `FORMPORTAL_ROOT` when set, else `~/Library/Application Support/studio.formai.portal` on macOS, `%LOCALAPPDATA%\FormPortal` on Windows and `$XDG_DATA_HOME/formportal` (or `~/.local/share/formportal`) elsewhere. It is read on the computer the form.md server runs on.
- The server lists them at `GET /api/extensions` and serves their files at `GET /api/extensions/files/{id}/{version}/{path}`, with the contract's CSP and no access to anything outside the folder.

## Limits

- `minimum_app` is compared with form.md's version (0.53.0 in this release; extensions need 0.50.0 or newer).
- A headless command has 120 s; through MCP the call returns an error after 60 s even if the command is still running in the window.
- Consent is stored in the browser per extension id, version and permission set; another browser or the desktop app asks again.
