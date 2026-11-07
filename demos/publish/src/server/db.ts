import * as Y from "yjs";
import type { CommentThread, Member, Page, PageVersion, Share, ShareLink, Site, User, Workspace } from "../../../../web/src/engine/types";

export interface Person {
  id: string;
  name: string;
  email: string;
  color: string;
}

export interface Version {
  meta: PageVersion;
  state: string;
}

export const people: Person[] = [];
export const pages: Page[] = [];
export const threads = new Map<string, CommentThread[]>();
export const versions = new Map<string, Version[]>();
export const sites = new Map<string, Site>();
export const passwords = new Map<string, string>();
export const shares = new Map<string, Share[]>();
export const links = new Map<string, ShareLink[]>();
export const favorites: string[] = [];
export const visits = new Map<string, number>();
export const state = { me: null as User | null, workspace: null as Workspace | null, members: [] as Member[] };

const docs = new Map<string, Y.Doc>();

export function serverDoc(id: string): Y.Doc {
  let d = docs.get(id);
  if (!d) {
    d = new Y.Doc();
    docs.set(id, d);
  }
  return d;
}

export function dropDoc(id: string): void {
  docs.get(id)?.destroy();
  docs.delete(id);
}

export function person(id: string): Person {
  return people.find((p) => p.id === id) ?? people[0]!;
}
