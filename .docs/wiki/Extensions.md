# Extensions

form.md loads extensions made to the Form Extension Contract (API 1), the same contract every Form app uses. FormPortal installs them, form.ext makes them, formai.studio lists them. An extension adds panels and commands; it runs in a sandboxed frame and reaches the page only through form.md's own commands, so it can never do what form.md itself cannot.

The API an extension sees in form.md (the document snapshot, the `doc.*` commands, the events) is in `docs/EXTENSIONS.md` in the repository.

## Where they come from

The server reads two files under the FormPortal root on the computer it runs on:

| File | Written by | Source shown |
|---|---|---|
| `extensions/formmd/enabled.json` | FormPortal, when you install or enable an extension | Catalogue or File |
| `extensions/formmd/dev.json` | form.ext's Run in App, for a folder you are working on | Development |

The FormPortal root is `FORMPORTAL_ROOT` when set; otherwise `~/Library/Application Support/studio.formai.portal` on macOS, `%LOCALAPPDATA%\FormPortal` on Windows and `~/.local/share/formportal` (or `$XDG_DATA_HOME/formportal`) on Linux. The window checks both files every 2 seconds while it has focus and on Reload Extensions, and updates only the rows that changed.

An extension is refused, with the reason in its row, when its manifest is not valid, is made for another app, needs another API version, is native code, has its entry page outside the folder, or needs a newer form.md.

## In the window

- **Extensions** in the sidebar foot opens the Extensions menu: one item per contributed command, then Extensions Panel, Reload Extensions and Manage in FormPortal. The desktop app also has an Extensions menu in the menu bar.
- **Extensions panel**: a pane on the right listing each extension with its version, source, permission chips and state (Ready, Needs Permission, Refused). Open shows its panels; a development row shows its folder.
- Panels an extension contributes open as tabs of that pane, or as dialogs.
- Find a Command (Mod+Shift+P) lists every contributed command under Extensions.

The first time a version of an extension runs, form.md asks you to allow what it asks for (read and change the open document, open files you choose, save files where you choose). The answer is kept in this browser for that version; a new version that asks for more asks again.

## Routes

| Route | Does |
|---|---|
| `GET /api/extensions` | the extensions with their state, permissions, panels and commands, plus a `stamp` of the two files |
| `GET /api/extensions/stamp` | only the stamp, for cheap polling |
| `GET /api/extensions/files/{id}/{version}/{path}` | an extension's own files, with the contract's Content-Security-Policy, `nosniff` and `no-store`; anything outside the folder, any symlink and any other file type answers 404 |

The list needs a signed-in session or token. The file route needs none, because the sandboxed frame has no origin to send cookies from; it serves only the package files of extensions that are ready.

## Agents

| Tool or command | Does |
|---|---|
| MCP `extensions_list` | the extensions as the open window sees them |
| MCP `extension_command` | runs a contributed command in the open window, after the person allows the extension |
| `./form-md cli extensions` | the same list from a terminal |
| `./form-md cli extension ID COMMAND key=value` | runs a contributed command |

They go through the window like `window_command_run` (see [[Terminal-and-Claude-Code]]), so a window must be open for the same account, and consent is asked there.
