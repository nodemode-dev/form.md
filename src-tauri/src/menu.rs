use std::sync::Mutex;

use tauri::menu::{Menu, MenuBuilder, MenuEvent, MenuItemBuilder, SubmenuBuilder};
use tauri::{AppHandle, Manager, Runtime};

pub struct Zoom(pub Mutex<f64>);

const ZOOM_STEPS: [f64; 9] = [0.67, 0.75, 0.8, 0.9, 1.0, 1.1, 1.25, 1.5, 1.75];

fn item<R: Runtime>(app: &AppHandle<R>, id: &str, label: &str, keys: Option<&str>) -> tauri::Result<tauri::menu::MenuItem<R>> {
    let mut b = MenuItemBuilder::with_id(id, label);
    if let Some(k) = keys {
        b = b.accelerator(k);
    }
    b.build(app)
}

pub fn build<R: Runtime>(app: &AppHandle<R>) -> tauri::Result<Menu<R>> {
    let app_menu = SubmenuBuilder::new(app, "form.md")
        .about(None)
        .separator()
        .item(&item(app, "cmd:prefs.open", "Settings...", Some("CmdOrCtrl+,"))?)
        .separator()
        .services()
        .separator()
        .hide()
        .hide_others()
        .show_all()
        .separator()
        .quit()
        .build()?;
    let file = SubmenuBuilder::new(app, "File")
        .item(&item(app, "cmd:page.new", "New Page", Some("CmdOrCtrl+N"))?)
        .item(&item(app, "cmd:templates.open", "New from Template...", None)?)
        .item(&item(app, "cmd:transfer.import", "Import...", None)?)
        .separator()
        .item(&item(app, "cmd:search.open", "Search...", Some("CmdOrCtrl+P"))?)
        .item(&item(app, "cmd:page.copyLink", "Copy Page Link", Some("CmdOrCtrl+L"))?)
        .separator()
        .close_window()
        .build()?;
    let edit = SubmenuBuilder::new(app, "Edit")
        .undo()
        .redo()
        .separator()
        .cut()
        .copy()
        .paste()
        .select_all()
        .separator()
        .item(&item(app, "cmd:page.find", "Find and Replace...", Some("CmdOrCtrl+F"))?)
        .build()?;
    let view = SubmenuBuilder::new(app, "View")
        .item(&item(app, "cmd:shell.toggleSidebar", "Toggle Sidebar", Some("CmdOrCtrl+\\"))?)
        .item(&item(app, "cmd:shell.theme.toggle", "Toggle Dark Mode", Some("CmdOrCtrl+Shift+L"))?)
        .separator()
        .item(&item(app, "cmd:nav.back", "Back", Some("CmdOrCtrl+["))?)
        .item(&item(app, "cmd:nav.forward", "Forward", Some("CmdOrCtrl+]"))?)
        .item(&item(app, "cmd:route.home", "Home", None)?)
        .item(&item(app, "cmd:library.open", "Library", None)?)
        .separator()
        .item(&item(app, "zoom:in", "Zoom In", Some("CmdOrCtrl+="))?)
        .item(&item(app, "zoom:out", "Zoom Out", Some("CmdOrCtrl+-"))?)
        .item(&item(app, "zoom:reset", "Actual Size", Some("CmdOrCtrl+0"))?)
        .separator()
        .item(&item(app, "app:reload", "Reload", Some("CmdOrCtrl+R"))?)
        .fullscreen()
        .build()?;
    let extensions = SubmenuBuilder::new(app, "Extensions")
        .item(&item(app, "cmd:extensions.menu", "Extension Commands...", None)?)
        .separator()
        .item(&item(app, "cmd:extensions.panel", "Extensions Panel", None)?)
        .item(&item(app, "cmd:extensions.reload", "Reload Extensions", None)?)
        .item(&item(app, "cmd:extensions.manage", "Manage in FormPortal", None)?)
        .build()?;
    let window = SubmenuBuilder::new(app, "Window")
        .minimize()
        .maximize()
        .separator()
        .bring_all_to_front()
        .build()?;
    let help = SubmenuBuilder::new(app, "Help")
        .item(&item(app, "cmd:palette.open", "Find a Command...", Some("CmdOrCtrl+Shift+P"))?)
        .item(&item(app, "cmd:shell.showShortcuts", "Keyboard Shortcuts", None)?)
        .separator()
        .item(&item(app, "cmd:demo.editor", "Demo: Editor and Databases", None)?)
        .item(&item(app, "cmd:demo.views", "Demo: Database Views", None)?)
        .item(&item(app, "cmd:demo.publish", "Demo: Writing and Publishing", None)?)
        .separator()
        .item(&item(app, "app:notices", "Third-Party Notices", None)?)
        .build()?;
    MenuBuilder::new(app)
        .items(&[&app_menu, &file, &edit, &view, &extensions, &window, &help])
        .build()
}

fn step(current: f64, up: bool) -> f64 {
    if up {
        ZOOM_STEPS.iter().copied().find(|z| *z > current + 0.001).unwrap_or(current)
    } else {
        ZOOM_STEPS.iter().rev().copied().find(|z| *z < current - 0.001).unwrap_or(current)
    }
}

pub fn handle<R: Runtime>(app: &AppHandle<R>, event: MenuEvent) {
    let Some(window) = app.get_webview_window("main") else {
        return;
    };
    let id = event.id().as_ref();
    if let Some(command) = id.strip_prefix("cmd:") {
        let js = format!("window.__formmdRun && window.__formmdRun({command:?})");
        let _ = window.eval(js);
        return;
    }
    match id {
        "zoom:in" | "zoom:out" | "zoom:reset" => {
            let zoom = app.state::<Zoom>();
            let mut z = zoom.0.lock().unwrap();
            *z = match id {
                "zoom:in" => step(*z, true),
                "zoom:out" => step(*z, false),
                _ => 1.0,
            };
            let _ = window.set_zoom(*z);
        }
        "app:reload" => {
            let _ = window.eval("location.reload()");
        }
        "app:notices" => {
            let _ = window.eval("window.open(location.origin + '/notices')");
        }
        _ => {}
    }
}
