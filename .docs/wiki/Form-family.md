# Form family

form.md is one of three apps that can run side by side on one computer: form.md for documents and databases, formlit for photos (library and raw developer) and formvis for layered image editing. Each keeps its own data and its own MCP server. form.md does not proxy their tools; it tells people and agents which app fits a task and how to reach it.

## Layout

| Path | What lives there |
|---|---|
| `internal/family/family.go` | The descriptor, the sibling lookups, the task router and the two routes below |
| `internal/mcp/tools_family.go` | The `family_list` and `family_route` MCP tools |
| `internal/config/config.go` | `FORMMD_FAMILY`, where the siblings run |

## Discovery

- `GET /.well-known/form-family.json` is public and describes this instance: `product`, `version`, `purpose`, `objects`, `rest`, `openapi`, `mcp`, `open` (the page URL pattern, `{id}` replaced by a page id), `auth` and `family` (the other product names). The URLs use `instance.public_url` when it is set, and the response is sent with `Cache-Control: no-store`.
- `GET /api/family` needs a signed-in person or a token and returns `self` (the descriptor) and `members`, one per sibling.

Neither sibling serves a descriptor of its own, so form.md reads what they already expose:

| App | Default address | Health | MCP status | Tools |
|---|---|---|---|---|
| formlit | `http://127.0.0.1:8390` | `GET /api/health` (`version`) | `GET /api/mcp/status` (`endpoint`) | `POST /mcp` `tools/list` |
| formvis | `http://127.0.0.1:8190` | `GET /version` (`version`) | `GET /api/mcp/status` (`endpoint`) | `POST /mcp` `tools/list` |

A member reports `reachable`, `version`, `mcp`, `tool_count`, `connect` (how an agent attaches: each app's own `mcp` command or plugin) and `problem` when it is not running or its tool list could not be read. Results are kept for `family.cache_seconds` (30); each request waits at most `family.timeout_ms` (2000) and follows no redirects. Requests that arrive while a sibling is being looked up wait for that lookup instead of starting their own, and a lookup cut short because its caller went away is not cached.

## Routing a task

`family_route` takes a task in plain words and ranks all three apps, form.md included. Each app scores `family.vocabulary_weight` points per task word that matches its own vocabulary (photo, raw, cull and develop for formlit; layer, mask, background and generative for formvis; page, wiki, database and task for form.md), plus `family.tool_weight` per word matched in each of its best `family.max_tools` live tools, which are also returned by name. Every `family.*` number lives in the contextual-compression skill's `policy.json`, with the lookup timeout and cache time.

`context_pack` adds a **Form family** note when a sibling scores at least `family.pack_min_score` and more than form.md for the task: the app, whether it is running, how to connect and up to `family.pack_max_tools` matching tools, marked untrusted like the rest of the pack. For a task with no word from either sibling's vocabulary, `context_pack` does not contact the siblings (`family_route` and `GET /api/family` always do). The skill's `family-routing` sub-skill tells agents what to do with the note. The agent then connects to that app's MCP server itself; the siblings ask in their own window before an outside agent's first call.

## Safety

- Lookups only go to loopback addresses (`127.0.0.0/8`, `::1`, `localhost`). A non-loopback entry in `FORMMD_FAMILY` is ignored, and so is an MCP endpoint a sibling reports on any host and port other than the base form.md looked it up at; form.md then uses `base/mcp`.
- What a sibling returns is cut down before it reaches an agent: tool names that are not 1 to 64 of `A-Z a-z 0-9 _ . -` are dropped, versions are cut to 32 characters and descriptions to 200 characters, on character boundaries.
- `FORMMD_FAMILY=off` stops every lookup; members then carry the reason in `problem`.
- form.md sends no credentials to the siblings. It identifies itself with `X-Formlit-Agent: form.md` and `X-Formvis-Client: form.md`.

## Related

[[API-and-agents]] · [[Configuration]] · [[Context-and-knowledge]]
