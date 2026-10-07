export type InstanceRole = "owner" | "admin" | "member";
export type WorkspaceRole = "admin" | "member" | "viewer";
export type AgentAccess = "open" | "strict" | "closed";

export interface Prefs {
  lang: string;
  region: string;
  time_zone: string;
  hour_cycle: "" | "h12" | "h23";
  week_start: "" | "monday" | "sunday" | "saturday";
  notes_mode: boolean;
  system_font: boolean;
}

export interface User {
  id: string;
  email: string;
  name: string;
  color: string;
  avatar: string;
  role: InstanceRole;
  email_verified: boolean;
  disabled: boolean;
  totp_enabled: boolean;
  prefs: Prefs;
  created_at: number;
}

export interface Workspace {
  id: string;
  name: string;
  icon: string;
  image: string;
  role: WorkspaceRole;
  personal: boolean;
  auto_join: boolean;
  sidebar_mode: "split" | "tree";
  agent_access: AgentAccess;
  has_rules: boolean;
  proposal_waiting: boolean;
  member_count: number;
  break_glass?: boolean;
  owner_id: string;
  rules_updated_at: number;
}

export interface Me {
  user: User;
  credential: { kind: "session" | "token" | "oauth"; scope: "read" | "write" };
  instance: { name: string };
  workspaces: Workspace[];
  withheld_workspaces: number;
  can_create_workspaces: boolean;
}

export interface SSOProvider {
  id: string;
  name: string;
}

export interface Boot {
  setup?: boolean;
  me?: Me;
  sso?: SSOProvider[];
  signup?: "open" | "closed";
  instance_name?: string;
  workspace_id?: string;
}

export interface Member {
  user_id: string;
  name: string;
  email: string;
  color: string;
  avatar: string;
  role: WorkspaceRole;
  joined_at: number;
  disabled: boolean;
}

export interface Invite {
  id: string;
  email: string;
  role: WorkspaceRole;
  workspace_id: string;
  created_by: string;
  created_at: number;
  expires_at: number;
}

export interface InviteInfo {
  workspace: { name: string; icon: string };
  inviter: string;
  role: WorkspaceRole;
  email: string;
  account_exists: boolean;
  signed_in_as?: string;
}

export interface Token {
  id: string;
  name: string;
  prefix: string;
  scope: "read" | "write";
  all_workspaces: boolean;
  workspaces: string[];
  created_at: number;
  last_used_at: number;
  last_used_ip: string;
}

export interface SessionInfo {
  id: string;
  current: boolean;
  created_at: number;
  last_seen_at: number;
  user_agent: string;
  ip: string;
}

export interface Rules {
  rules: string;
  updated_at: number;
  proposal?: string;
  proposal_by?: string;
  proposal_by_name?: string;
  proposal_at?: number;
  can_edit: boolean;
}

export interface AdminUser {
  id: string;
  email: string;
  name: string;
  color: string;
  avatar: string;
  role: InstanceRole;
  disabled: boolean;
  totp_enabled: boolean;
  email_verified: boolean;
  created_at: number;
  last_seen_at: number;
  workspaces: number;
}

export interface DeletionPlan {
  user_id: string;
  name: string;
  email: string;
  purge: { id: string; name: string }[];
  promote: { id: string; name: string; user_id: string; user_name: string }[];
  hand_to_owner: { id: string; name: string }[];
}

export interface BreakGlassGrant {
  id: string;
  workspace_id: string;
  workspace_name: string;
  reason: string;
  created_at: number;
  expires_at: number;
}

export interface EmergencyGrant {
  id: string;
  user_name: string;
  reason: string;
  created_at: number;
  expires_at: number;
  revoked_at: number;
}

export interface AccessWorkspace {
  id: string;
  name: string;
  icon: string;
  auto_join: boolean;
  agent_access: AgentAccess;
  members: { user_id: string; role: WorkspaceRole }[];
}

export interface StrandedWorkspace {
  id: string;
  name: string;
  created_at: number;
  members: number;
}

export type SettingsMap = Record<string, string>;
