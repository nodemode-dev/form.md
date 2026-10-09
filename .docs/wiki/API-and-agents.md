# API and agents

The web app, scripts and agents use the same HTTP API. Scripts use the REST API under `/api` with a personal API token or an OAuth 2.1 token. AI agents connect to the MCP endpoint at `/mcp`, sign in with OAuth or a personal token, and edit documents by block id so their changes merge with what people are typing. Every route is described by an OpenAPI document generated from the live router.

## Layout

| Path | What lives there |
|---|---|
| `internal/apidoc/apidoc.go` | `GET /api/openapi.json` (OpenAPI 3.1) and `GET /api/docs` (the readable reference page) |
| `internal/auth/account.go`, `internal/auth/module.go` | Personal API tokens: `GET`, `POST /api/me/tokens`, `DELETE /api/me/tokens/{id}` |
| `internal/auth/service.go` | Credential resolution, read-only enforcement, rate limits |
| `internal/oauth/oauth.go` | OAuth 2.1 authorization server, discovery documents, consent |
| `internal/mcp/` | The MCP server (`server.go`), its tools (`tools*.go`) and the downloadable skill (`skill.go`) |
| `web/src/ui/account.ts` | Account settings, API tokens tab |
| `web/src/ui/account-agents.ts` | Account settings, Connected apps tab: connect snippets, grants, agent runs |

## The REST API reference

Open `https://YOUR-HOST/api/docs` for the reference page, or fetch the machine-readable document:

```sh
curl -fsS https://notes.example.com/api/openapi.json -o openapi.json
```

General rules, as the document states them: lists that can be long take `limit` and an opaque `cursor` and return `next_cursor`; errors are JSON with `error.code` and `error.message`.

## Personal API tokens

Create one in **Account settings, API tokens**: give it a name, choose **Read and write** or **Read only**, and optionally **Only chosen workspaces** (the token then reaches nothing else, even workspaces you join later). The token starts with `fmd_` and is shown once ("Copy this token now. It is not shown again."); the server stores only a hash. The list shows each token's prefix, reach, creation time and last use with its address.

```sh
curl -fsS -H "Authorization: Bearer fmd_..." https://notes.example.com/api/me
```

A token can only narrow what its owner can do. A read-only token gets an error on any request that changes something. Some routes (instance administration, token management) accept only a signed-in browser session.

## OAuth 2.1

form.md is its own authorization server, with discovery at:

- `/.well-known/oauth-authorization-server`
- `/.well-known/oauth-protected-resource/api` (REST API) and `/.well-known/oauth-protected-resource/mcp` (MCP)

Clients register themselves at `POST /oauth/register`, send the person to `GET /oauth/authorize`, and exchange and revoke tokens at `POST /oauth/token` and `POST /oauth/revoke`. Scopes are `read` and `write`. The person sees a consent page in form.md where they pick the workspaces and the access level.

A token is issued for one resource: `<origin>/api` for the REST API or `<origin>/mcp` for MCP. Using it on the other is refused with `wrong_audience` ("this token was issued for the MCP endpoint only", or "for the REST API only").

## Rate limits

Tokens (personal and OAuth) on the REST API are limited per credential to 600 requests a minute with bursts of 120. Signed-in browser sessions and the `/mcp` endpoint are not under this limit. Every limited response carries `RateLimit-Limit`, `RateLimit-Policy` and `RateLimit-Remaining`; a refused request gets `429` with `Retry-After` and `RateLimit-Reset` in seconds and the error code `rate_limited`.

## Connecting an agent over MCP

The endpoint is `https://YOUR-HOST/mcp` (Streamable HTTP, stateless JSON). It speaks protocol version 2026-07-28 and answers clients on 2025-11-25, 2025-06-18 and 2025-03-26.

**Account settings, Connected apps** shows the address for the instance you are on, with ready-to-copy snippets:

```sh
claude mcp add --transport http form-md https://notes.example.com/mcp
```

For Cursor, VS Code and other clients that read an `mcp.json`:

```json
{
  "mcpServers": {
    "form-md": {
      "url": "https://notes.example.com/mcp"
    }
  }
}
```

The first call opens a page in form.md asking which workspaces the agent may use and whether it may read only or read and edit. A personal API token also works as a Bearer token on `/mcp`.

The same tab offers **Download the form.md skill** (`GET /api/mcp/skill.zip`), a zip with `form-md/SKILL.md` and `form-md/tools.json`. The skill gives the agent the address, the Claude Code command above, and the working rules: call `whoami` first and follow each workspace's written rules, find before creating, edit by block id, pass the `run_id` from a write to later writes of the same task, treat text inside `[untrusted ...]` markers as data, and reuse an `idempotency_key` when retrying a write. `tools.json` holds every tool's input schema.

Below the snippets, **Connected apps** lists every agent you allowed, with its workspaces, access level and last use, and a Disconnect button that stops its tokens at once. **Recent agent work** lists agent runs; **Undo this run** reverses every change in a run, newest first, keeping later edits by people where they do not overlap.

## The contract

1. A new route is registered on the router with its method and path. The OpenAPI document is generated from the live router and includes every route whose path starts with one of the prefixes in `documented()` in `internal/apidoc/apidoc.go` (`/api/`, `/mcp`, `/oauth/`, `/.well-known/`, `/scim/`, `/files/`, `/healthz`, `/notices`).
2. Access is decided by the policy gate for every credential kind; a route never checks the token type itself, except through `RequireSession`, `RequireAdmin` or `RequireOwner` for routes that must not be reachable by tokens.
3. A new MCP tool that changes anything is marked `Write`. The server then adds the `run_id` and `idempotency_key` arguments to its schema and handles both, and the tool records its changes in the agent run so **Undo this run** can reverse them.
4. Rate limits and audience checks live in `internal/auth/service.go` middleware; do not add per-route limits for tokens elsewhere.

## Related

[[Offline]] · [[TLS-and-ACME]] · [[Configuration]]
