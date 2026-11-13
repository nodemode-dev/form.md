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


def seed_traffic(context, base, views):
    boot = context.request.get(f"{base}/api/me").json()
    ws = boot.get("workspace") or (boot.get("workspaces") or [{}])[0]
    made = context.request.post(f"{base}/api/pages", data={"workspace_id": ws.get("id", ""), "title": "Site traffic", "kind": "collection", "extra": {"view": "table"}})
    assert made.ok, f"seed database: {made.status} {made.text()}"
    col = made.json()["page"]["id"]
    schema = context.request.post(f"{base}/api/collections/{col}/schema", data={"ops": [{"op": "add", "prop": {"id": "", "name": "Day", "type": "date"}}]}).json()["schema"]
    day = next(p["id"] for p in schema if p["name"] == "Day")
    today = time.localtime()
    for title, back in (("Visit today", 0), ("Visit ten days ago", 10), ("Visit forty days ago", 40)):
        d = time.strftime("%Y-%m-%d", time.localtime(time.mktime(today) - back * 86400))
        row = context.request.post(f"{base}/api/collections/{col}/rows", data={"title": title, "values": {day: {"start": d}}})
        assert row.ok, f"seed row: {row.status} {row.text()}"
    for i in range(views - 1):
        v = context.request.post(f"{base}/api/collections/{col}/views", data={"name": f"Traffic by referrer {i + 2:02d}", "type": "table"})
        assert v.ok, f"seed view: {v.status} {v.text()}"
    return col


def test_many_view_tabs_stay_inside_the_database_and_stay_reachable(engine):
    from playwright.sync_api import sync_playwright

    errors = []
    with sync_playwright() as p:
        browser = p.chromium.launch()
        context = browser.new_context(viewport={"width": 1280, "height": 900})
        sign_in_and_seed(context, engine)
        col = seed_traffic(context, engine, 15)
        page = context.new_page()
        page.on("pageerror", lambda e: errors.append(str(e)))
        page.goto(f"{engine}/p/{col}", wait_until="load", timeout=120_000)
        page.wait_for_selector(".db__tabs [data-view][aria-selected='true']", timeout=60_000)
        page.wait_for_selector(".db__tab--more:not([hidden])", timeout=10_000)
        box = page.evaluate("""() => {
            const r = (s) => document.querySelector(s).getBoundingClientRect();
            const tabs = document.querySelector('.db__tabs');
            return { db: r('.db').right, bar: r('.db__bar').right, views: r('.db__views').right, scroll: tabs.scrollWidth, client: tabs.clientWidth, count: document.querySelectorAll('.db__tabs [data-view]').length };
        }""")
        page.click(".db__tab--more")
        items = page.locator(".menu .menu__item")
        listed = items.count()
        last = items.nth(listed - 1).inner_text().strip()
        items.nth(listed - 1).click()
        page.wait_for_selector(".db__tabs [data-view][aria-selected='true']:has-text('Traffic by referrer 15')", timeout=10_000)
        shown = page.evaluate("""() => {
            const t = document.querySelector('.db__tabs').getBoundingClientRect();
            const a = document.querySelector('.db__tabs [aria-selected="true"]').getBoundingClientRect();
            return a.left >= t.left - 1 && a.right <= t.right + 1;
        }""")
        browser.close()
    assert box["count"] == 15, box
    assert box["bar"] <= box["db"] + 1 and box["views"] <= box["db"] + 1, f"the tab strip overflows the database: {box}"
    assert box["scroll"] > box["client"], f"15 tabs should scroll inside the strip at 1280 px: {box}"
    assert listed >= 1 and last == "Traffic by referrer 15", f"the more menu lists {listed} views, last {last!r}"
    assert shown, "the picked view's tab is scrolled into sight"
    assert not errors, f"page errors: {errors}"


def test_date_range_narrows_the_view_and_is_kept_per_viewer(engine):
    from playwright.sync_api import sync_playwright

    errors, bodies = [], []
    with sync_playwright() as p:
        browser = p.chromium.launch()
        context = browser.new_context(viewport={"width": 1440, "height": 900})
        sign_in_and_seed(context, engine)
        col = seed_traffic(context, engine, 1)
        page = context.new_page()
        page.on("pageerror", lambda e: errors.append(str(e)))
        page.on("request", lambda r: bodies.append(r.post_data or "") if r.url.endswith("/query") else None)
        page.goto(f"{engine}/p/{col}", wait_until="load", timeout=120_000)
        page.wait_for_selector(".db__stage :text('Visit forty days ago')", timeout=60_000)
        page.click(".db__tool[aria-label='Date range']")
        page.click(".menu .menu__item:has-text('Last 7 days')")
        page.wait_for_selector(".db__stage :text('Visit forty days ago')", state="detached", timeout=10_000)
        narrowed = page.locator(".db__stage").inner_text()
        label = page.locator(".db__tool[aria-label='Date range'] .db__tool-label").inner_text()
        page.add_init_script("""
            window.__leak = false;
            new MutationObserver(() => {
              if (document.body && document.body.innerText.includes('Visit forty days ago')) window.__leak = true;
            }).observe(document, { subtree: true, childList: true, characterData: true });
        """)
        page.reload(wait_until="load")
        page.wait_for_selector(".db__stage :text('Visit today')", timeout=60_000)
        page.wait_for_timeout(1500)
        leaked = page.evaluate("window.__leak")
        kept = page.locator(".db__tool[aria-label='Date range'] .db__tool-label").inner_text()
        page.click(".db__tool[aria-label='Date range']")
        page.click(".menu .menu__item:has-text('Custom range')")
        page.fill(".daterange input[aria-label='Start date']", time.strftime("%Y-%m-%d", time.localtime(time.time() - 45 * 86400)))
        page.press(".daterange input[aria-label='End date']", "Enter")
        page.wait_for_selector(".db__stage :text('Visit forty days ago')", timeout=10_000)
        browser.close()
    assert "Visit today" in narrowed and "Visit ten days ago" not in narrowed, narrowed
    assert label == "Last 7 days" and kept == "Last 7 days", (label, kept)
    assert not leaked, "rows outside the kept range were painted on reload"
    assert any('"date_range"' in b for b in bodies), bodies
    assert not errors, f"page errors: {errors}"
