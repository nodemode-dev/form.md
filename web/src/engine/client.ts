import { type FetchFn, request, stream, upload } from "./api";
import type {
  AccessWorkspace,
  AdminUser,
  AIAdmin,
  AIAsk,
  AIChoices,
  AgentChat,
  AgentGrant,
  AgentMessage,
  AgentRun,
  AgentState,
  CustomAgent,
  CustomAgentRun,
  BreakGlassGrant,
  Collection,
  CollectionView,
  CommentListing,
  CommentThread,
  CSVPreview,
  PageComment,
  FormDef,
  Property,
  QueryInput,
  QueryResult,
  RowData,
  RowDetail,
  SchemaOp,
  ViewConfig,
  ViewType,
  DeletionPlan,
  EmergencyGrant,
  Favorite,
  FileInfo,
  Group,
  InboxItem,
  LinkPreview,
  NotifyPrefs,
  PageVersion,
  PageAnalytics,
  WorkspaceAnalytics,
  Webhook,
  WebhookDelivery,
  WebhookInput,
  AuditPage,
  AuditQuery,
  ScimToken,
  Site,
  SiteInput,
  SiteStatus,
  FilePurpose,
  Invite,
  InviteInfo,
  Me,
  Member,
  Page,
  PageDetail,
  Prefs,
  RecentPage,
  Rules,
  SearchQuery,
  SearchResult,
  SessionInfo,
  SettingsMap,
  Share,
  SharedPage,
  ShareLink,
  Sharing,
  StrandedWorkspace,
  Suggestion,
  TagInfo,
  TeamAccess,
  TeamMember,
  Teamspace,
  Token,
  TrashItem,
  Visibility,
  Workspace,
  WorkspaceRole,
} from "./types";

export interface PageChanges {
  title?: string;
  icon?: string;
  cover?: string;
  description?: string;
  tags?: string[];
  visibility?: Visibility;
}

export interface MoveTarget {
  workspace_id?: string;
  parent_id?: string;
  teamspace_id?: string;
  before_id?: string;
  after_id?: string;
}

export interface TeamChanges {
  name?: string;
  icon?: string;
  description?: string;
  access?: TeamAccess;
}

export interface ShareRequest {
  kind?: "user" | "group";
  subject_id?: string;
  email?: string;
  level: string;
  expires_at?: number;
}

function auditParams(q: AuditQuery): string {
  const p = new URLSearchParams();
  for (const [k, v] of Object.entries(q)) if (v !== undefined && v !== "" && v !== 0) p.set(k, String(v));
  return p.toString();
}

export function createClient(f: FetchFn) {
  const get = <T>(path: string) => request<T>(f, "GET", path);
  const post = <T>(path: string, body: unknown = {}) => request<T>(f, "POST", path, body);
  const patch = <T>(path: string, body: unknown) => request<T>(f, "PATCH", path, body);
  const put = <T>(path: string, body: unknown) => request<T>(f, "PUT", path, body);
  const del = (path: string) => request<void>(f, "DELETE", path);
  const enc = encodeURIComponent;

  return {
    agent: {
      state: (ws: string) => get<AgentState>(`/api/agent?workspace=${enc(ws)}`),
      setPrefs: (ws: string, prefs: { instructions_page: string; memory: boolean }) => put<void>("/api/agent/prefs", { workspace_id: ws, ...prefs }),
      forget: (id: string) => del(`/api/agent/memory/${enc(id)}`),
      newChat: (ws: string) => post<{ chat: AgentChat }>("/api/agent/chats", { workspace_id: ws }),
      chat: (id: string) => get<{ chat: AgentChat; messages: AgentMessage[] }>(`/api/agent/chats/${enc(id)}`),
      deleteChat: (id: string) => del(`/api/agent/chats/${enc(id)}`),
      send: (id: string, body: { text: string; page_id?: string }, onEvent: (event: string, data: unknown) => void, cancel?: AbortSignal) =>
        stream(f, `/api/agent/chats/${enc(id)}/messages`, body, (e) => onEvent(e.event, e.data), cancel),
      list: (ws: string) => get<{ agents: CustomAgent[] }>(`/api/workspaces/${enc(ws)}/agents`),
      create: (ws: string, body: Partial<CustomAgent>) => post<{ agent: CustomAgent }>(`/api/workspaces/${enc(ws)}/agents`, body),
      update: (id: string, body: Partial<CustomAgent>) => patch<{ agent: CustomAgent }>(`/api/agents/${enc(id)}`, body),
      remove: (id: string) => del(`/api/agents/${enc(id)}`),
      run: (id: string) => post<{ chat: AgentChat }>(`/api/agents/${enc(id)}/run`),
      runs: (id: string) => get<{ runs: CustomAgentRun[] }>(`/api/agents/${enc(id)}/runs`),
    },
    ai: {
      models: (workspaceId = "") => get<AIChoices>(`/api/ai/models${workspaceId ? `?workspace=${enc(workspaceId)}` : ""}`),
      complete: (body: AIAsk, onText: (t: string) => void, cancel?: AbortSignal) =>
        stream(f, "/api/ai/complete", body, (e) => e.event === "delta" && onText((e.data as { t: string }).t), cancel),
      admin: () => get<AIAdmin>("/api/admin/ai"),
      saveSettings: (body: { enabled?: boolean; default_model?: string; transcribe_model?: string; limit_per_hour?: number }) => put<void>("/api/admin/ai", body),
      addProvider: (body: { kind: string; name?: string; base_url?: string; api_key?: string }) => post<{ id: string; models: number; warning?: string }>("/api/admin/ai/providers", body),
      updateProvider: (id: string, body: { name?: string; base_url?: string; api_key?: string; enabled?: boolean }) => patch<void>(`/api/admin/ai/providers/${enc(id)}`, body),
      removeProvider: (id: string) => del(`/api/admin/ai/providers/${enc(id)}`),
      refreshModels: (id: string) => post<{ models: number }>(`/api/admin/ai/providers/${enc(id)}/refresh`),
      setModel: (id: string, body: { model: string; label?: string; enabled?: boolean; audio?: boolean }) => put<void>(`/api/admin/ai/providers/${enc(id)}/models`, body),
      test: (model: string) => post<{ reply: string }>("/api/admin/ai/test", { model }),
      transcribe: (form: FormData) => upload<{ text: string }>(f, "/api/ai/transcribe", form, 600000),
      answer: (workspaceId: string, question: string, onEvent: (event: string, data: unknown) => void, cancel?: AbortSignal) =>
        stream(f, "/api/ai/answer", { workspace_id: workspaceId, question }, (e) => onEvent(e.event, e.data), cancel),
      setWorkspace: (workspaceId: string, enabled: boolean) => put<void>(`/api/workspaces/${enc(workspaceId)}/ai`, { enabled }),
    },
    auth: {
      policy: () =>
        get<{ setup_required: boolean; signup: "open" | "invite"; sso: { id: string; name: string }[]; instance_name: string }>("/api/auth/policy"),
      setup: (name: string, email: string, password: string) => post<Me>("/api/setup", { name, email, password }),
      login: (email: string, password: string, code = "", recovery = "") => post<Me>("/api/auth/login", { email, password, code, recovery }),
      signup: (name: string, email: string, password: string) => post<Me>("/api/auth/signup", { name, email, password }),
      logout: () => post<void>("/api/auth/logout"),
      invite: (token: string) => get<InviteInfo>(`/api/invites/${enc(token)}`),
      acceptInvite: (token: string, body: { name?: string; email?: string; password?: string; code?: string; recovery?: string }) =>
        post<{ workspace_id: string; page_id?: string }>(`/api/invites/${enc(token)}/accept`, body),
    },
    me: {
      get: () => get<Me>("/api/me"),
      update: (body: { name?: string; color?: string; avatar?: string; prefs?: Prefs }) => patch<Me>("/api/me", body),
      password: (current: string, next: string) => post<void>("/api/me/password", { current, next }),
      email: (password: string, email: string) => post<Me>("/api/me/email", { password, email }),
      totpSetup: () => post<{ secret: string; uri: string; qr_svg: string }>("/api/me/totp/setup"),
      totpConfirm: (code: string) => post<{ recovery_codes: string[] }>("/api/me/totp/confirm", { code }),
      totpDisable: (code: string, recovery = "") => post<void>("/api/me/totp/disable", { code, recovery }),
      tokens: () => get<{ tokens: Token[] }>("/api/me/tokens"),
      createToken: (name: string, scope: "read" | "write", workspaces: string[] | null) =>
        post<{ id: string; token: string }>("/api/me/tokens", { name, scope, workspaces }),
      deleteToken: (id: string) => del(`/api/me/tokens/${enc(id)}`),
      sessions: () => get<{ sessions: SessionInfo[] }>("/api/me/sessions"),
      endSession: (id: string) => del(`/api/me/sessions/${enc(id)}`),
      grants: () => get<{ grants: AgentGrant[] }>("/api/me/grants"),
      revokeGrant: (id: string) => del(`/api/me/grants/${enc(id)}`),
      consent: (request: string, approve: boolean, scope: "read" | "write", workspaces: string[]) =>
        post<{ redirect: string }>("/api/oauth/consent", { request, approve, scope, workspaces }),
      runs: (agent = "") => get<{ runs: AgentRun[] }>(`/api/agent-runs${agent ? `?agent=${enc(agent)}` : ""}`),
      undoRun: (id: string) => post<{ undone: number }>(`/api/agent-runs/${enc(id)}/undo`),
    },
    workspaces: {
      list: () => get<{ workspaces: Workspace[]; withheld: number }>("/api/workspaces"),
      create: (name: string, icon = "") => post<{ workspace: Workspace }>("/api/workspaces", { name, icon }),
      update: (id: string, body: Partial<Pick<Workspace, "name" | "icon" | "image" | "sidebar_mode" | "agent_access" | "auto_join">>) =>
        patch<{ workspace: Workspace }>(`/api/workspaces/${enc(id)}`, body),
      remove: (id: string) => del(`/api/workspaces/${enc(id)}`),
      members: (id: string) => get<{ members: Member[] }>(`/api/workspaces/${enc(id)}/members`),
      addMember: (id: string, email: string, role: WorkspaceRole) => post<void>(`/api/workspaces/${enc(id)}/members`, { email, role }),
      setRole: (id: string, user: string, role: WorkspaceRole) => patch<void>(`/api/workspaces/${enc(id)}/members/${enc(user)}`, { role }),
      removeMember: (id: string, user: string) => del(`/api/workspaces/${enc(id)}/members/${enc(user)}`),
      invites: (id: string) => get<{ invites: Invite[] }>(`/api/workspaces/${enc(id)}/invites`),
      invite: (id: string, email: string, role: WorkspaceRole) =>
        post<{ invite: Invite; url: string; emailed: boolean }>(`/api/workspaces/${enc(id)}/invites`, { email, role }),
      cancelInvite: (id: string, invite: string) => del(`/api/workspaces/${enc(id)}/invites/${enc(invite)}`),
      rules: (id: string) => get<Rules>(`/api/workspaces/${enc(id)}/rules`),
      setRules: (id: string, rules: string) => put<void>(`/api/workspaces/${enc(id)}/rules`, { rules }),
      dismissProposal: (id: string) => del(`/api/workspaces/${enc(id)}/rules/proposal`),
      emergency: (id: string) => get<{ grants: EmergencyGrant[] }>(`/api/workspaces/${enc(id)}/emergency`),
      endEmergency: (id: string, grant: string) => del(`/api/workspaces/${enc(id)}/emergency/${enc(grant)}`),
      pages: (id: string) => get<{ pages: Page[] }>(`/api/workspaces/${enc(id)}/pages`),
      trash: (id: string, q = "") => get<{ items: TrashItem[] }>(`/api/workspaces/${enc(id)}/trash${q ? `?q=${enc(q)}` : ""}`),
      emptyTrash: (id: string) => request<{ deleted: number }>(f, "DELETE", `/api/workspaces/${enc(id)}/trash`),
      tags: (id: string) => get<{ tags: TagInfo[] }>(`/api/workspaces/${enc(id)}/tags`),
      setTagColor: (id: string, tag: string, color: string) => put<void>(`/api/workspaces/${enc(id)}/tags/${enc(tag)}`, { color }),
      files: (id: string) => get<{ files: FileInfo[] }>(`/api/workspaces/${enc(id)}/files`),
      teamspaces: (id: string) => get<{ teamspaces: Teamspace[] }>(`/api/workspaces/${enc(id)}/teamspaces`),
      createTeamspace: (id: string, body: TeamChanges) => post<{ teamspace: Teamspace }>(`/api/workspaces/${enc(id)}/teamspaces`, body),
      groups: (id: string) => get<{ groups: Group[] }>(`/api/workspaces/${enc(id)}/groups`),
      createGroup: (id: string, name: string) => post<{ group: Group }>(`/api/workspaces/${enc(id)}/groups`, { name }),
    },
    teamspaces: {
      update: (id: string, body: TeamChanges) => patch<{ teamspace: Teamspace }>(`/api/teamspaces/${enc(id)}`, body),
      remove: (id: string) => del(`/api/teamspaces/${enc(id)}`),
      members: (id: string) => get<{ members: TeamMember[] }>(`/api/teamspaces/${enc(id)}/members`),
      setMember: (id: string, user: string, role: "owner" | "member") => put<void>(`/api/teamspaces/${enc(id)}/members/${enc(user)}`, { role }),
      removeMember: (id: string, user: string) => del(`/api/teamspaces/${enc(id)}/members/${enc(user)}`),
    },
    groups: {
      rename: (id: string, name: string) => patch<{ group: Group }>(`/api/groups/${enc(id)}`, { name }),
      remove: (id: string) => del(`/api/groups/${enc(id)}`),
      members: (id: string) => get<{ members: TeamMember[] }>(`/api/groups/${enc(id)}/members`),
      addMember: (id: string, user: string) => put<void>(`/api/groups/${enc(id)}/members/${enc(user)}`, {}),
      removeMember: (id: string, user: string) => del(`/api/groups/${enc(id)}/members/${enc(user)}`),
    },
    comments: {
      list: (pageId: string) => get<CommentListing>(`/api/pages/${enc(pageId)}/comments`),
      create: (pageId: string, body: { anchor: "page" | "text" | "block"; start?: string; end?: string; quote?: string; body: string }) =>
        post<{ thread: CommentThread }>(`/api/pages/${enc(pageId)}/comments`, body),
      reply: (threadId: string, body: string) => post<{ comment: PageComment }>(`/api/threads/${enc(threadId)}/comments`, { body }),
      resolve: (threadId: string, resolved: boolean) => put<void>(`/api/threads/${enc(threadId)}/resolved`, { resolved }),
      edit: (id: string, body: string) => patch<void>(`/api/comments/${enc(id)}`, { body }),
      remove: (id: string) => del(`/api/comments/${enc(id)}`),
      react: (id: string, icon: string, on: boolean) => (on ? put<void>(`/api/comments/${enc(id)}/reactions/${enc(icon)}`, {}) : del(`/api/comments/${enc(id)}/reactions/${enc(icon)}`)),
      suggest: (pageId: string, body: { start: string; end: string; original: string; text: string }) => post<{ suggestion: Suggestion }>(`/api/pages/${enc(pageId)}/suggestions`, body),
      revise: (id: string, body: { start: string; end: string; original: string; text: string }) => patch<void>(`/api/suggestions/${enc(id)}`, body),
      decide: (id: string, accept: boolean) => put<void>(`/api/suggestions/${enc(id)}/decision`, { accept }),
    },
    history: {
      list: (pageId: string) => get<{ versions: PageVersion[] }>(`/api/pages/${enc(pageId)}/versions`),
      get: (id: string) => get<{ version: PageVersion; state: string }>(`/api/versions/${enc(id)}`),
      save: (pageId: string, beforeRestore = false) => post<{ id: string }>(`/api/pages/${enc(pageId)}/versions${beforeRestore ? "?kind=restore" : ""}`),
      restored: (id: string) => post<void>(`/api/versions/${enc(id)}/restored`),
    },
    webhooks: {
      list: (workspaceId: string) => get<{ webhooks: Webhook[] }>(`/api/workspaces/${enc(workspaceId)}/webhooks`),
      create: (workspaceId: string, input: WebhookInput) => post<{ webhook: Webhook; secret: string }>(`/api/workspaces/${enc(workspaceId)}/webhooks`, input),
      update: (id: string, input: WebhookInput) => patch<{ webhook: Webhook }>(`/api/webhooks/${enc(id)}`, input),
      remove: (id: string) => del(`/api/webhooks/${enc(id)}`),
      rotate: (id: string) => post<{ secret: string }>(`/api/webhooks/${enc(id)}/rotate`),
      test: (id: string) => post<void>(`/api/webhooks/${enc(id)}/test`),
      deliveries: (id: string) => get<{ deliveries: WebhookDelivery[] }>(`/api/webhooks/${enc(id)}/deliveries`),
      retry: (deliveryId: string) => post<void>(`/api/deliveries/${enc(deliveryId)}/retry`),
    },
    scim: {
      tokens: (workspaceId: string) => get<{ tokens: ScimToken[]; base_url: string }>(`/api/workspaces/${enc(workspaceId)}/scim`),
      create: (workspaceId: string, label: string) => post<{ token: string; info: ScimToken }>(`/api/workspaces/${enc(workspaceId)}/scim`, { label }),
      revoke: (id: string) => del(`/api/scim-tokens/${enc(id)}`),
    },
    audit: {
      list: (workspaceId: string, q: AuditQuery) => get<AuditPage>(`/api/workspaces/${enc(workspaceId)}/audit?${auditParams(q)}`),
      csvUrl: (workspaceId: string, q: AuditQuery) => `/api/workspaces/${enc(workspaceId)}/audit.csv?${auditParams(q)}`,
    },
    analytics: {
      page: (pageId: string, days: number) => get<PageAnalytics>(`/api/pages/${enc(pageId)}/analytics?days=${days}`),
      workspace: (workspaceId: string, days: number) => get<WorkspaceAnalytics>(`/api/workspaces/${enc(workspaceId)}/analytics?days=${days}`),
      setHistory: (workspaceId: string, enabled: boolean) => put<void>(`/api/workspaces/${enc(workspaceId)}/view-history`, { enabled }),
      prefs: () => get<{ record: boolean }>("/api/me/analytics"),
      setPrefs: (record: boolean) => put<void>("/api/me/analytics", { record }),
    },
    sites: {
      status: (pageId: string) => get<SiteStatus>(`/api/pages/${enc(pageId)}/site`),
      publish: (pageId: string, input: SiteInput) => put<{ site: Site }>(`/api/pages/${enc(pageId)}/site`, input),
      unpublish: (pageId: string) => del(`/api/pages/${enc(pageId)}/site`),
      list: (workspaceId: string) => get<{ sites: Site[]; publishing: boolean; can_manage: boolean }>(`/api/workspaces/${enc(workspaceId)}/sites`),
      setPublishing: (workspaceId: string, enabled: boolean) => put<void>(`/api/workspaces/${enc(workspaceId)}/publishing`, { enabled }),
    },
    inbox: {
      list: (filter: "all" | "unread" | "archived", before = 0) => get<{ items: InboxItem[]; unread: number }>(`/api/inbox?filter=${filter}${before ? `&before=${before}` : ""}`),
      count: () => get<{ unread: number }>("/api/inbox/count"),
      mark: (body: { ids?: string[]; all?: boolean; read?: boolean; archived?: boolean; seen?: boolean }) => post<void>("/api/inbox/mark", body),
      prefs: () => get<NotifyPrefs>("/api/me/notifications"),
      setPrefs: (email: NotifyPrefs["email"]) => put<void>("/api/me/notifications", { email }),
      subscribe: (sub: { endpoint: string; keys: { p256dh: string; auth: string } }) => post<void>("/api/me/push", sub),
      unsubscribe: (endpoint: string) => request<void>(f, "DELETE", "/api/me/push", { endpoint }),
      testPush: () => post<{ sent: number }>("/api/me/push/test"),
    },
    sharing: {
      get: (pageId: string) => get<Sharing>(`/api/pages/${enc(pageId)}/shares`),
      share: (pageId: string, body: ShareRequest) => put<{ share?: Share; invite_url?: string; emailed: boolean }>(`/api/pages/${enc(pageId)}/shares`, body),
      update: (id: string, level: string, expiresAt = 0) => patch<void>(`/api/shares/${enc(id)}`, { level, expires_at: expiresAt }),
      remove: (id: string) => del(`/api/shares/${enc(id)}`),
      createLink: (pageId: string, body: { level: "view" | "comment"; expires_at?: number; password?: string }) => post<{ link: ShareLink }>(`/api/pages/${enc(pageId)}/links`, body),
      revokeLink: (id: string) => del(`/api/links/${enc(id)}`),
      cancelInvite: (id: string) => del(`/api/guest-invites/${enc(id)}`),
      shared: () => get<{ pages: SharedPage[] }>("/api/shared"),
    },
    pages: {
      get: (id: string) => get<{ page: PageDetail }>(`/api/pages/${enc(id)}`),
      create: (body: { workspace_id: string; parent_id?: string; teamspace_id?: string; after_id?: string; title?: string; icon?: string }) => post<{ page: Page }>("/api/pages", body),
      update: (id: string, body: PageChanges) => patch<{ page: Page }>(`/api/pages/${enc(id)}`, body),
      move: (id: string, body: MoveTarget) => post<{ page: Page }>(`/api/pages/${enc(id)}/move`, body),
      duplicate: (id: string, title = "") => post<{ page: Page }>(`/api/pages/${enc(id)}/duplicate`, title ? { title } : {}),
      trash: (id: string) => request<{ trashed: string[] }>(f, "DELETE", `/api/pages/${enc(id)}`),
      restore: (id: string) => post<{ page: Page }>(`/api/pages/${enc(id)}/restore`),
      deleteForever: (id: string) => del(`/api/pages/${enc(id)}/forever`),
      visit: (id: string) => post<void>(`/api/pages/${enc(id)}/visit`),
      recent: (limit = 50) => get<{ pages: RecentPage[] }>(`/api/recent?limit=${limit}`),
      search: (q: SearchQuery, cancel?: AbortSignal) => {
        const p = new URLSearchParams({ q: q.q, workspace_id: q.workspace_id });
        if (q.in === "title") p.set("in", "title");
        if (q.creator?.length) p.set("creator", q.creator.join(","));
        if (q.parent) p.set("parent", q.parent);
        if (q.date) p.set("date", q.date);
        if (q.from) p.set("from", String(q.from));
        if (q.to) p.set("to", String(q.to));
        if (q.sort && q.sort !== "best") p.set("sort", q.sort);
        if (q.limit) p.set("limit", String(q.limit));
        if (q.offset) p.set("offset", String(q.offset));
        return request<SearchResult>(f, "GET", `/api/search?${p.toString()}`, undefined, 15000, cancel);
      },
      backlinks: (id: string) => get<{ pages: Page[] }>(`/api/pages/${enc(id)}/backlinks`),
      createCollection: (body: { workspace_id: string; parent_id?: string; after_id?: string; title?: string; inline?: boolean; view?: ViewType }) =>
        post<{ page: Page }>("/api/pages", {
          workspace_id: body.workspace_id,
          parent_id: body.parent_id,
          after_id: body.after_id,
          title: body.title ?? "",
          kind: "collection",
          extra: { inline: !!body.inline, view: body.view ?? "table" },
        }),
    },
    collections: {
      get: (id: string, view?: string) =>
        get<{ collection: Collection; first?: { view_id: string; result: QueryResult } }>(`/api/collections/${enc(id)}${view !== undefined ? `?view=${enc(view)}` : ""}`),
      schema: (id: string, ops: SchemaOp[]) => post<{ schema: Property[]; epoch: number }>(`/api/collections/${enc(id)}/schema`, { ops }),
      formula: (id: string, expr: string) => post<{ ok: boolean; error?: string; pos: number; props?: string[] }>(`/api/collections/${enc(id)}/formula`, { expr }),
      autofill: (id: string, property: string, rows: string[], onEvent: (event: string, data: { total?: number; row?: string; ok?: boolean; filled?: number }) => void, cancel?: AbortSignal) =>
        stream(f, `/api/collections/${enc(id)}/autofill`, { property, rows }, (e) => onEvent(e.event, e.data as { total?: number; row?: string; ok?: boolean; filled?: number }), cancel),
      query: (id: string, input: QueryInput, cancel?: AbortSignal) => request<QueryResult>(f, "POST", `/api/collections/${enc(id)}/query`, input, 15000, cancel),
      search: (id: string, q: string, limit = 50) => get<{ rows: { id: string; title: string; icon: string }[] }>(`/api/collections/${enc(id)}/search?q=${enc(q)}&limit=${limit}`),
      createRow: (id: string, body: { title?: string; values?: Record<string, unknown>; before_id?: string; after_id?: string; top?: boolean }) =>
        post<{ row: RowData; epoch: number }>(`/api/collections/${enc(id)}/rows`, body),
      deleteRows: (id: string, ids: string[]) => post<{ trashed: number }>(`/api/collections/${enc(id)}/rows/delete`, { ids }),
      row: (id: string) => get<RowDetail>(`/api/rows/${enc(id)}`),
      updateRow: (id: string, body: { title?: string; values?: Record<string, unknown>; before_id?: string; after_id?: string; move?: boolean }) =>
        patch<{ row: RowData; epoch: number }>(`/api/rows/${enc(id)}`, body),
      createView: (id: string, body: { name?: string; type?: ViewType; config?: ViewConfig; after_id?: string }) =>
        post<{ view: CollectionView }>(`/api/collections/${enc(id)}/views`, body),
      updateView: (id: string, body: { name?: string; type?: ViewType; config?: ViewConfig }) => patch<{ view: CollectionView }>(`/api/views/${enc(id)}`, body),
      deleteView: (id: string) => del(`/api/views/${enc(id)}`),
      moveView: (id: string, afterId: string) => post<void>(`/api/views/${enc(id)}/move`, { after_id: afterId }),
      duplicateView: (id: string) => post<{ view: CollectionView }>(`/api/views/${enc(id)}/duplicate`),
      exportUrl: (viewId: string) => `/api/views/${enc(viewId)}/export.csv`,
      previewCSV: (file: File) => {
        const form = new FormData();
        form.append("file", file);
        return upload<CSVPreview>(f, "/api/csv/preview", form);
      },
      importCSV: (body: { token: string; types: string[]; title?: string; workspace_id?: string; parent_id?: string; collection_id?: string }) =>
        post<{ collection_id: string; rows: number }>("/api/csv/import", body),
      form: (viewId: string) => get<{ form: FormDef }>(`/api/views/${enc(viewId)}/form`),
      submit: (viewId: string, answers: Record<string, unknown>) => post<{ row_id: string }>(`/api/views/${enc(viewId)}/form`, { answers }),
      formLink: (viewId: string, pub: boolean) => put<{ token: string }>(`/api/views/${enc(viewId)}/form/link`, { public: pub }),
      publicForm: (token: string) => get<{ form: FormDef }>(`/api/public/forms/${enc(token)}`),
      publicSubmit: (token: string, answers: Record<string, unknown>) => post<void>(`/api/public/forms/${enc(token)}`, { answers }),
    },
    favorites: {
      list: () => get<{ favorites: Favorite[] }>("/api/favorites"),
      add: (id: string, body: { after_id?: string; top?: boolean } = {}) => put<void>(`/api/favorites/${enc(id)}`, body),
      remove: (id: string) => del(`/api/favorites/${enc(id)}`),
    },
    files: {
      upload: (file: Blob, purpose: FilePurpose, target: { workspace_id?: string; page_id?: string; name?: string } = {}) => {
        const form = new FormData();
        form.append("purpose", purpose);
        if (target.workspace_id) form.append("workspace_id", target.workspace_id);
        if (target.page_id) form.append("page_id", target.page_id);
        if (target.name) form.append("name", target.name);
        form.append("file", file);
        return upload<{ file: FileInfo }>(f, "/api/files", form);
      },
      importUrl: (url: string, pageId: string) => request<{ file: FileInfo }>(f, "POST", "/api/files/import", { url, page_id: pageId }, 60000),
      unfurl: (url: string, pageId: string) => request<LinkPreview>(f, "GET", `/api/unfurl?url=${enc(url)}&page_id=${enc(pageId)}`, undefined, 30000),
    },
    admin: {
      users: () => get<{ users: AdminUser[] }>("/api/admin/users"),
      createUser: (body: { name: string; email: string; password: string; admin: boolean }) => post<{ id: string }>("/api/admin/users", body),
      setAccess: (userId: string, workspaceId: string, role: WorkspaceRole | "") =>
        put<void>(`/api/admin/users/${enc(userId)}/workspaces/${enc(workspaceId)}`, { role }),
      updateUser: (id: string, body: { role?: "admin" | "member"; disabled?: boolean }) => patch<void>(`/api/admin/users/${enc(id)}`, body),
      resetTOTP: (id: string) => post<void>(`/api/admin/users/${enc(id)}/reset-2fa`),
      setPassword: (id: string, password: string) => post<void>(`/api/admin/users/${enc(id)}/password`, { password }),
      deletionPlan: (id: string) => get<DeletionPlan>(`/api/admin/users/${enc(id)}/deletion-plan`),
      deleteUser: (id: string) => del(`/api/admin/users/${enc(id)}`),
      transfer: (userId: string, password: string) => post<void>("/api/admin/ownership", { user_id: userId, password }),
      breakGlass: (workspaceId: string, reason: string) => post<{ id: string; expires_at: number }>("/api/admin/break-glass", { workspace_id: workspaceId, reason }),
      grants: () => get<{ grants: BreakGlassGrant[] }>("/api/admin/break-glass"),
      revokeGrant: (id: string) => del(`/api/admin/break-glass/${enc(id)}`),
      stranded: () => get<{ workspaces: StrandedWorkspace[] }>("/api/admin/stranded"),
      adopt: (id: string) => post<void>(`/api/admin/stranded/${enc(id)}/adopt`),
      deleteStranded: (id: string) => del(`/api/admin/stranded/${enc(id)}`),
      access: () => get<{ workspaces: AccessWorkspace[] }>("/api/admin/access"),
      settings: () => get<{ settings: SettingsMap; secret_set: string; mail_enabled: boolean }>("/api/admin/settings"),
      saveSettings: (changes: SettingsMap) => put<{ settings: SettingsMap; secret_set: string; mail_enabled: boolean }>("/api/admin/settings", changes),
      testMail: (to: string) => post<void>("/api/admin/mail/test", { to }),
    },
  };
}

export type Client = ReturnType<typeof createClient>;
