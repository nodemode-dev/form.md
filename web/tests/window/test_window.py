import os
import shutil
import socket
import subprocess
import sys
import tempfile
import time
import urllib.request
from pathlib import Path

import pytest

WINDOW_DIR = Path(__file__).resolve().parent
REPO = WINDOW_DIR.parents[2]

APP_ID = "form.md"
ENGINE_DIR = REPO
ENGINE_CMD = [os.environ.get("FORMMD_TEST_BIN", str(REPO / "bin" / "form-md"))]
DESKTOP_KEY = "window-smoke-desktop-key-0123456789abcdef"
ENGINE_ENV = {"FORMMD_DATA": "{data}", "FORMMD_ADDR": "127.0.0.1:{port}", "FORMMD_DESKTOP_KEY": DESKTOP_KEY, "FORMMD_FAMILY": "off", "FORMMD_UPDATE_URL": "http://127.0.0.1:1/none"}
CLEAR_ENV_PREFIX = "FORMMD_"
DIST = REPO / "web" / "dist"
HEALTH_PATH = "/healthz"
PAGES = [
    ("/home", ":root[data-auth='app'][data-route='home'] .sidebar"),
    ("/p/{page}", ".ProseMirror[contenteditable='true']"),
    ("/sandbox.html", "#stage"),
]
SETTLE_MS = 3000
CLS_LIMIT = 0.0
IGNORED_REQUESTS = []

CPU_SLOWDOWN = 4
NETWORK = {"offline": False, "latency": 150, "downloadThroughput": 1_600_000 / 8, "uploadThroughput": 750_000 / 8}

CLS_PROBE = """
window.__cls = 0;
window.__shifts = [];
new PerformanceObserver((list) => {
  for (const e of list.getEntries()) {
    if (e.hadRecentInput) continue;
    window.__cls += e.value;
    const box = (r) => [r.x, r.y, r.width, r.height].map(Math.round).join(',');
    const who = (e.sources || []).map((s) => (s.node ? `${s.node.nodeName}.${String(s.node.className || '').trim().split(/\\s+/)[0]}` : '?') + `[${box(s.previousRect)}>${box(s.currentRect)}]`);
    window.__shifts.push(`${Math.round(e.startTime)}ms ${e.value.toExponential(2)} ${who.join(',')}`);
  }
}).observe({ type: 'layout-shift', buffered: true });
"""


def free_port():
    with socket.socket() as s:
        s.bind(("127.0.0.1", 0))
        return s.getsockname()[1]


@pytest.fixture(scope="module")
def engine():
    assert (DIST / "index.html").is_file(), f"build the window first: {DIST} has no index.html (npm run build in web)"
    assert Path(ENGINE_CMD[0]).is_file(), f"build the server first: {ENGINE_CMD[0]} (go build -o bin/form-md ./cmd/form-md)"
    port = free_port()
    sandbox = Path(tempfile.mkdtemp(prefix="formmd-window-"))
    (sandbox / "data").mkdir()
    (sandbox / "home").mkdir()
    values = {"port": str(port), "data": str(sandbox / "data"), "dist": str(DIST)}
    env = {k: v for k, v in os.environ.items() if not k.startswith(CLEAR_ENV_PREFIX)}
    env.update({k: v.format(**values) for k, v in ENGINE_ENV.items()})
    env["HOME"] = str(sandbox / "home")
    proc = subprocess.Popen([a.format(**values) for a in ENGINE_CMD], cwd=ENGINE_DIR, env=env, stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
    base = f"http://127.0.0.1:{port}"
    deadline = time.monotonic() + 90
    while time.monotonic() < deadline and proc.poll() is None:
        try:
            with urllib.request.urlopen(base + HEALTH_PATH, timeout=2) as r:
                if r.status == 200:
                    break
        except OSError:
            time.sleep(0.25)
    else:
        proc.kill()
        out = proc.communicate(timeout=5)[0].decode("utf-8", "replace")
        raise RuntimeError(f"engine did not answer {HEALTH_PATH}:\n{out[-4000:]}")
    yield base
    proc.terminate()
    try:
        proc.wait(timeout=10)
    except subprocess.TimeoutExpired:
        proc.kill()
    shutil.rmtree(sandbox, ignore_errors=True)


def ignored(url):
    return any(part in url for part in IGNORED_REQUESTS)


def sign_in_and_seed(context, base):
    entered = context.request.get(f"{base}/desktop/enter?key={DESKTOP_KEY}")
    assert entered.ok, f"the desktop entry did not sign the owner in: {entered.status}"
    boot = context.request.get(f"{base}/api/me").json()
    ws = boot.get("workspace") or (boot.get("workspaces") or [{}])[0]
    made = context.request.post(f"{base}/api/pages", data={"workspace_id": ws.get("id", ""), "title": "Window smoke", "kind": "doc"})
    assert made.ok, f"seed page: {made.status} {made.text()}"
    return made.json()["page"]["id"]


@pytest.mark.parametrize("path,ready", PAGES)
def test_window_loads_throttled_without_shift_errors_or_refusals(engine, path, ready):
    from playwright.sync_api import sync_playwright

    errors, failed = [], []
    with sync_playwright() as p:
        browser = p.chromium.launch()
        context = browser.new_context(viewport={"width": 1440, "height": 900})
        page_id = sign_in_and_seed(context, engine)
        page = context.new_page()
        page.add_init_script(CLS_PROBE)
        cdp = context.new_cdp_session(page)
        cdp.send("Emulation.setCPUThrottlingRate", {"rate": CPU_SLOWDOWN})
        cdp.send("Network.enable")
        cdp.send("Network.emulateNetworkConditions", NETWORK)
        page.on("console", lambda m: errors.append(m.text) if m.type == "error" else None)
        page.on("pageerror", lambda e: errors.append(str(e)))
        no_content = set()
        page.on("requestfailed", lambda r: failed.append(f"{r.url} {r.failure}") if not ignored(r.url) and r not in no_content else None)
        page.on("response", lambda r: no_content.add(r.request) if r.status == 204 else None)
        page.on("response", lambda r: failed.append(f"{r.url} {r.status}") if r.status >= 400 and not ignored(r.url) else None)
        page.goto(engine + path.format(page=page_id), wait_until="load", timeout=120_000)
        mounted = page.wait_for_selector(ready, state="attached", timeout=120_000)
        try:
            page.wait_for_load_state("networkidle", timeout=20_000)
        except Exception:
            pass
        page.wait_for_timeout(SETTLE_MS)
        cls = page.evaluate("window.__cls")
        shifts = page.evaluate("window.__shifts")
        title = page.title()
        url = page.url
        browser.close()
    assert mounted, f"the window did not mount ({ready}) at {url}"
    assert title, "the window has a title"
    assert not [f for f in failed if " 403" in f], f"requests refused by the guard: {failed}"
    assert not failed, f"failed requests: {failed}"
    assert not errors, f"console errors: {errors}"
    assert cls <= CLS_LIMIT, f"layout shift {cls}: {shifts}"


DEMOS = [("editor", ".sidebar"), ("views", ".sidebar"), ("publish", ".ProseMirror")]


def test_every_demo_opens_in_the_window_from_home_and_help(engine):
    from playwright.sync_api import sync_playwright

    errors = []
    with sync_playwright() as p:
        browser = p.chromium.launch()
        context = browser.new_context(viewport={"width": 1440, "height": 900})
        sign_in_and_seed(context, engine)
        page = context.new_page()
        page.on("pageerror", lambda e: errors.append(str(e)))
        page.goto(engine + "/home", wait_until="load", timeout=120_000)
        page.wait_for_selector("[data-ref='home-demos-sec'][data-ready]", state="attached", timeout=60_000)
        for via in ("home", "help"):
            for demo, ready in DEMOS:
                if via == "home":
                    page.click(f"[data-ref='home-demos-grid'] [data-demo='{demo}']")
                else:
                    page.click("[data-ref='help-btn']")
                    page.click(f".menu [data-menu-command='demo.{demo}']")
                page.wait_for_selector(f"[data-demo-stage='{demo}'] iframe[data-demo-frame='{demo}']", timeout=30_000)
                frame = page.frame_locator(f"iframe[data-demo-frame='{demo}']")
                frame.locator(ready).first.wait_for(timeout=60_000)
                page.click(f"[data-demo-stage='{demo}'] [data-demo-close]")
                page.wait_for_selector("[data-demo-stage]", state="detached", timeout=10_000)
        browser.close()
    assert not errors, f"page errors: {errors}"
