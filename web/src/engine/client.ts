import { type FetchFn, request, upload } from "./api";
import type {
  AccessWorkspace,
  AdminUser,
  BreakGlassGrant,
  Collection,
  CollectionView,
  CSVPreview,
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
  StrandedWorkspace,
  TagInfo,
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
  before_id?: string;
  after_id?: string;
}

export function createClient(f: FetchFn) {
  const get = <T>(path: string) => request<T>(f, "GET", path);
  const post = <T>(path: string, body: unknown = {}) => request<T>(f, "POST", path, body);
  const patch = <T>(path: string, body: unknown) => request<T>(f, "PATCH", path, body);
  const put = <T>(path: string, body: unknown) => request<T>(f, "PUT", path, body);
  const del = (path: string) => request<void>(f, "DELETE", path);
  const enc = encodeURIComponent;

  return {
    auth: {
      policy: () =>
        get<{ setup_required: boolean; signup: "open" | "invite"; sso: { id: string; name: string }[]; instance_name: string }>("/api/auth/policy"),
      setup: (name: string, email: string, password: string) => post<Me>("/api/setup", { name, email, password }),
      login: (email: string, password: string, code = "", recovery = "") => post<Me>("/api/auth/login", { email, password, code, recovery }),
      signup: (name: string, email: string, password: string) => post<Me>("/api/auth/signup", { name, email, password }),
      logout: () => post<void>("/api/auth/logout"),
      invite: (token: string) => get<InviteInfo>(`/api/invites/${enc(token)}`),
      acceptInvite: (token: string, body: { name?: string; email?: string; password?: string; code?: string; recovery?: string }) =>
        post<{ workspace_id: string }>(`/api/invites/${enc(token)}/accept`, body),
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
    },
    pages: {
      get: (id: string) => get<{ page: PageDetail }>(`/api/pages/${enc(id)}`),
      create: (body: { workspace_id: string; parent_id?: string; after_id?: string; title?: string; icon?: string }) => post<{ page: Page }>("/api/pages", body),
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
