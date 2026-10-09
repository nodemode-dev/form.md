import { Fragment, type Node as PMNode } from "prosemirror-model";
import type { EditorView, NodeView, ViewMutationRecord } from "prosemirror-view";
import { ApiError } from "../../engine/api";
import { newBlockId, schema } from "../../engine/doc/schema";
import { t } from "../../engine/i18n";
import { h, setAttr, setText } from "../../ui/dom";
import { iconNode } from "../../ui/icons";
import { type MenuEntry, openMenu } from "../../ui/menu";
import { toast } from "../../ui/toast";
import type { EditorCtx } from "./context";
import { markdownReady, parseMarkdownSlice } from "./markdown";
import { registerNodeView } from "./nodeviews";

const SEGMENT_MS = 10 * 60 * 1000;
const MAX_MS = 4 * 60 * 60 * 1000;

function pickType(): string {
  for (const type of ["audio/webm;codecs=opus", "audio/webm", "audio/mp4", "audio/ogg;codecs=opus"]) if (typeof MediaRecorder !== "undefined" && MediaRecorder.isTypeSupported(type)) return type;
  return "";
}

class Recorder {
  readonly started = Date.now();
  private streams: MediaStream[] = [];
  private ctx = new AudioContext();
  private dest = this.ctx.createMediaStreamDestination();
  private analyser = this.ctx.createAnalyser();
  private buf = new Uint8Array(256);
  private rec: MediaRecorder | null = null;
  private rotate = 0;
  private limit = 0;
  readonly segments: Promise<Blob>[] = [];
  readonly type = pickType();
  tabAudio = false;
  onAutoStop: (() => void) | null = null;

  static async open(withTab: boolean): Promise<Recorder> {
    const r = new Recorder();
    try {
      const mic = await navigator.mediaDevices.getUserMedia({ audio: { echoCancellation: true, noiseSuppression: true } });
      r.add(mic);
      if (withTab) {
        const tab = await navigator.mediaDevices.getDisplayMedia({ video: true, audio: true });
        r.streams.push(tab);
        if (tab.getAudioTracks().length) {
          r.add(new MediaStream(tab.getAudioTracks()), false);
          r.tabAudio = true;
        }
      }
    } catch (e) {
      r.release();
      throw e;
    }
    r.next();
    r.rotate = window.setInterval(() => r.next(), SEGMENT_MS);
    r.limit = window.setTimeout(() => r.onAutoStop?.(), MAX_MS);
    return r;
  }

  private add(stream: MediaStream, keep = true): void {
    if (keep) this.streams.push(stream);
    const src = this.ctx.createMediaStreamSource(stream);
    src.connect(this.dest);
    src.connect(this.analyser);
  }

  private next(): void {
    if (this.rec && this.rec.state !== "inactive") this.rec.stop();
    const opts: MediaRecorderOptions = { audioBitsPerSecond: 32000 };
    if (this.type) opts.mimeType = this.type;
    const rec = new MediaRecorder(this.dest.stream, opts);
    const chunks: Blob[] = [];
    this.segments.push(
      new Promise<Blob>((resolve) => {
        rec.addEventListener("dataavailable", (e) => e.data.size && chunks.push(e.data));
        rec.addEventListener("stop", () => resolve(new Blob(chunks, { type: rec.mimeType || this.type || "audio/webm" })));
      }),
    );
    rec.start(1000);
    this.rec = rec;
  }

  level(): number {
    this.analyser.getByteTimeDomainData(this.buf);
    let sum = 0;
    for (const v of this.buf) sum += ((v - 128) / 128) ** 2;
    return Math.min(1, Math.sqrt(sum / this.buf.length) * 4);
  }

  private release(): void {
    clearInterval(this.rotate);
    clearTimeout(this.limit);
    for (const s of this.streams) for (const tr of s.getTracks()) tr.stop();
    void this.ctx.close().catch(() => undefined);
  }

  async stop(): Promise<Blob[]> {
    if (this.rec && this.rec.state !== "inactive") this.rec.stop();
    this.release();
    const blobs = await Promise.all(this.segments);
    return blobs.filter((b) => b.size > 0);
  }
}

interface Job {
  recorder: Recorder | null;
  blobs: Blob[];
  busy: boolean;
}

const jobs = new Map<string, Job>();

function clock(ms: number): string {
  const s = Math.max(0, Math.floor(ms / 1000));
  const m = Math.floor(s / 60);
  const hh = Math.floor(m / 60);
  const pad = (n: number) => String(n).padStart(2, "0");
  return hh ? `${hh}:${pad(m % 60)}:${pad(s % 60)}` : `${pad(m)}:${pad(s % 60)}`;
}

export function meetingMeta(state: string, started: number, duration: number): string {
  if (state === "recording") return t("Recording");
  if (state === "transcribing") return t("Transcribing");
  if (state === "summarizing") return t("Summarizing");
  if (state === "failed") return t("Transcription failed");
  if (started <= 0) return "";
  let out = new Date(started).toISOString().slice(0, 10);
  if (duration > 0) out += ` · ${Math.max(1, Math.round(duration / 60))} min`;
  return out;
}

export function newMeeting(): PMNode {
  return schema.nodes.meeting!.create({ id: newBlockId() }, [schema.nodes.paragraph!.create({ id: newBlockId() })]);
}

class MeetingView implements NodeView {
  readonly dom: HTMLElement;
  readonly contentDOM: HTMLElement;
  private meta: HTMLElement;
  private controls: HTMLElement;
  private level: HTMLElement;
  private frame = 0;
  private tick = 0;

  constructor(
    private node: PMNode,
    private readonly view: EditorView,
    private readonly getPos: () => number | undefined,
    private readonly ctx: EditorCtx,
  ) {
    this.meta = h("span", { class: "meeting__meta" });
    this.controls = h("span", { class: "meeting__controls" });
    this.level = h("span", { class: "meeting__level", attrs: { "aria-hidden": "true" } }, h("i"));
    const head = h("div", { class: "meeting__head", attrs: { contenteditable: "false" } }, h("span", { class: "meeting__icon" }, iconNode("audio")), h("span", { class: "meeting__title" }), this.meta, this.controls);
    this.contentDOM = h("div", { class: "meeting__body" });
    this.dom = h("div", { class: "meeting" }, head, this.contentDOM);
    this.controls.addEventListener("click", (e) => {
      const b = (e.target as Element).closest<HTMLButtonElement>("button[data-do]");
      if (b) this.act(b.dataset.do ?? "", b);
    });
    this.sync();
  }

  private id(): string {
    return String(this.node.attrs.id ?? "");
  }

  private job(): Job | undefined {
    return jobs.get(this.id());
  }

  private sync(): void {
    const a = this.node.attrs;
    setAttr(this.dom, "data-block", this.id() || null);
    setAttr(this.dom, "data-state", String(a.state ?? ""));
    setText(this.dom.querySelector(".meeting__title") as HTMLElement, String(a.title || t("Meeting notes")));
    const job = this.job();
    const rec = job?.recorder;
    if (rec) setText(this.meta, `${t("Recording")} ${clock(Date.now() - rec.started)}`);
    else setText(this.meta, meetingMeta(String(a.state ?? ""), Number(a.started) || 0, Number(a.duration) || 0));
    const want = rec ? "recording" : job?.busy ? "busy" : String(a.state ?? "") === "" && !Number(a.started) ? "idle" : ["recording", "transcribing", "summarizing"].includes(String(a.state)) ? "stale" : a.state === "failed" ? (job?.blobs.length ? "retry" : "stale") : "done";
    if (this.controls.dataset.mode !== want) {
      this.controls.dataset.mode = want;
      this.controls.replaceChildren();
      const btn = (label: string, what: string, primary = false) => h("button", { class: primary ? "meeting__go meeting__go--primary" : "meeting__go", attrs: { type: "button", "data-do": what }, text: label });
      if (want === "idle") this.controls.append(btn(t("Start"), "start", true));
      else if (want === "recording") this.controls.append(this.level, btn(t("Stop"), "stop", true));
      else if (want === "retry") this.controls.append(btn(t("Retry"), "retry", true), btn(t("Reset"), "reset"));
      else if (want === "stale") this.controls.append(btn(t("Reset"), "reset"));
    }
    if (rec && !this.tick) {
      this.tick = window.setInterval(() => this.sync(), 1000);
      const loop = () => {
        const r = this.job()?.recorder;
        if (!r) return;
        (this.level.firstElementChild as HTMLElement).style.transform = `scaleX(${r.level().toFixed(3)})`;
        this.frame = requestAnimationFrame(loop);
      };
      this.frame = requestAnimationFrame(loop);
    } else if (!rec && this.tick) {
      clearInterval(this.tick);
      cancelAnimationFrame(this.frame);
      this.tick = 0;
    }
  }

  private set(attrs: Record<string, unknown>): void {
    const pos = this.getPos();
    if (pos === undefined) return;
    const cur = this.view.state.doc.nodeAt(pos);
    if (!cur || cur.type !== this.node.type) return;
    this.view.dispatch(this.view.state.tr.setNodeMarkup(pos, undefined, { ...cur.attrs, ...attrs }));
  }

  private act(what: string, anchor: HTMLElement): void {
    if (!this.ctx.writable()) return;
    if (what === "start") {
      const entries: MenuEntry[] = [
        { label: t("Microphone"), icon: "audio", run: () => void this.start(false) },
        { label: t("Microphone and tab or system audio"), icon: "monitor", run: () => void this.start(true) },
      ];
      openMenu(anchor, entries, { side: "bottom", align: "end", label: t("Record from") });
    } else if (what === "stop") void this.finish();
    else if (what === "retry") void this.process();
    else if (what === "reset") {
      jobs.delete(this.id());
      this.set({ state: "", started: 0, duration: 0 });
    }
  }

  private async start(withTab: boolean): Promise<void> {
    if (!navigator.mediaDevices?.getUserMedia || typeof MediaRecorder === "undefined") {
      toast({ message: t("This browser cannot record audio."), tone: "warning" });
      return;
    }
    try {
      const rec = await Recorder.open(withTab);
      if (withTab && !rec.tabAudio) toast({ message: t("No tab or system audio was shared, so only the microphone is recorded."), tone: "warning" });
      rec.onAutoStop = () => void this.finish();
      jobs.set(this.id(), { recorder: rec, blobs: [], busy: false });
      this.set({ state: "recording", started: rec.started, duration: 0 });
      this.sync();
    } catch (e) {
      const denied = e instanceof DOMException && (e.name === "NotAllowedError" || e.name === "SecurityError");
      toast({ message: denied ? t("Recording needs permission to use the microphone.") : t("Recording could not start."), tone: "warning" });
    }
  }

  private async finish(): Promise<void> {
    const job = this.job();
    const rec = job?.recorder;
    if (!job || !rec) return;
    job.recorder = null;
    job.busy = true;
    const duration = Math.round((Date.now() - rec.started) / 1000);
    this.set({ state: "transcribing", duration });
    this.sync();
    job.blobs = await rec.stop();
    await this.process();
  }

  private async process(): Promise<void> {
    const job = this.job();
    if (!job || !job.blobs.length) {
      this.set({ state: "failed" });
      return;
    }
    job.busy = true;
    this.set({ state: "transcribing" });
    this.sync();
    try {
      const parts: string[] = [];
      for (const [i, blob] of job.blobs.entries()) {
        const form = new FormData();
        const ext = blob.type.includes("mp4") ? "m4a" : blob.type.includes("ogg") ? "ogg" : "webm";
        form.append("file", new File([blob], `meeting-${i + 1}.${ext}`, { type: blob.type.split(";")[0] || "audio/webm" }));
        form.append("page_id", this.ctx.pageId());
        form.append("workspace_id", this.ctx.workspaceId());
        const r = await this.ctx.engine.api.ai.transcribe(form);
        if (r.text) parts.push(r.text);
      }
      const transcript = parts.join("\n\n").trim();
      if (!transcript) throw new Error(t("Nothing was heard in the recording."));
      this.set({ state: "summarizing" });
      const pos = this.getPos();
      const notes = pos === undefined ? "" : this.ctx.markdownBetween(pos + 1, pos + this.node.nodeSize - 1);
      let summary = "";
      await this.ctx.engine.api.ai.complete({ action: "meeting", page_id: this.ctx.pageId(), workspace_id: this.ctx.workspaceId(), selection: transcript, before: notes }, (chunk) => {
        summary += chunk;
      });
      await this.place(summary.trim(), transcript);
      jobs.delete(this.id());
      this.set({ state: "done" });
    } catch (e) {
      job.busy = false;
      this.set({ state: "failed" });
      toast({ message: e instanceof ApiError || e instanceof Error ? e.message : t("Transcription failed."), tone: "warning" });
    }
    this.sync();
  }

  private async place(summary: string, transcript: string): Promise<void> {
    const ready = await markdownReady();
    const pos = this.getPos();
    if (pos === undefined) return;
    const state = this.view.state;
    const node = state.doc.nodeAt(pos);
    if (!node || node.type !== this.node.type) return;
    const end = pos + node.nodeSize - 1;
    const nodes: PMNode[] = [];
    if (summary && ready) parseMarkdownSlice(summary, state.doc.resolve(end)).content.forEach((n) => void (n.isBlock && nodes.push(n)));
    else if (summary) for (const line of summary.split(/\n{2,}/)) nodes.push(schema.nodes.paragraph!.create({ id: newBlockId() }, line.trim() ? schema.text(line.trim()) : null));
    const lines = transcript.split(/\n+/).map((l) => l.trim()).filter(Boolean).slice(0, 4000);
    const paras = lines.map((l) => schema.nodes.paragraph!.create({ id: newBlockId() }, schema.text(l)));
    nodes.push(schema.nodes.toggle!.create({ id: newBlockId() }, [schema.nodes.toggle_title!.create(null, schema.text(t("Transcript"))), ...paras]));
    const tr = state.tr;
    const only = node.childCount === 1 && node.firstChild?.type.name === "paragraph" && node.firstChild.content.size === 0;
    if (only) tr.replaceWith(pos + 1, end, Fragment.fromArray(nodes));
    else tr.insert(end, Fragment.fromArray(nodes));
    this.view.dispatch(tr);
  }

  update(node: PMNode): boolean {
    if (node.type !== this.node.type) return false;
    this.node = node;
    this.sync();
    return true;
  }

  stopEvent(e: Event): boolean {
    return !!(e.target as Element | null)?.closest?.(".meeting__head");
  }

  ignoreMutation(m: ViewMutationRecord): boolean {
    if (m.type === "selection") return false;
    return !this.contentDOM.contains(m.target);
  }

  destroy(): void {
    clearInterval(this.tick);
    cancelAnimationFrame(this.frame);
  }
}

registerNodeView("meeting", (ctx) => (node, view, getPos) => new MeetingView(node, view, getPos, ctx));
