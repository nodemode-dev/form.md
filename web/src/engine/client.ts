import { type FetchFn, request, upload } from "./api";
import type {
  AccessWorkspace,
  AdminUser,
  BreakGlassGrant,
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
