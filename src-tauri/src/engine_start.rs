use std::collections::VecDeque;
use std::path::{Path, PathBuf};
use std::time::Duration;

pub const APP: &str = "form.md";
pub const SHELL_ENV: &str = "FORMMD_SHELL_PID";
pub const ADDRESS_WAIT: Duration = Duration::from_secs(90);
pub const EXIT_WAIT: Duration = Duration::from_secs(1);
pub const BUSY_CODE: i32 = 2;
const LOG_NAME: &str = "form-md-server.log";
const KEPT_LINES: usize = 200;
const TAIL_LINES: usize = 12;
const LINE_CHARS: usize = 300;

#[derive(Debug, PartialEq)]
pub enum Failure {
    NotStarted(String),
    Stopped(Option<i32>),
    TimedOut,
    Window(String),
}

#[derive(Default)]
pub struct Tail(VecDeque<String>);

impl Tail {
    pub fn push(&mut self, line: &str) {
        let clean: String = line.chars().filter(|c| !c.is_control()).take(LINE_CHARS).collect();
        if clean.trim().is_empty() {
            return;
        }
        if self.0.len() == KEPT_LINES {
            self.0.pop_front();
        }
        self.0.push_back(clean);
    }

    pub fn lines(&self) -> Vec<String> {
        self.0.iter().cloned().collect()
    }
}

pub fn sentence(failure: &Failure) -> String {
    match failure {
        Failure::NotStarted(reason) => format!("The {APP} server could not start: {reason}"),
        Failure::Stopped(Some(BUSY_CODE)) => format!("{APP} could not start: another copy of its server holds its data folder."),
        Failure::Stopped(Some(code)) => format!("{APP} could not start: its server stopped with code {code} before the window could open."),
        Failure::Stopped(None) => format!("{APP} could not start: its server stopped before the window could open."),
        Failure::TimedOut => format!(
            "{APP} could not start: it waited {} seconds for its server to say which address it serves on and that did not happen.",
            ADDRESS_WAIT.as_secs()
        ),
        Failure::Window(reason) => format!("{APP} could not open its window: {reason}"),
    }
}

pub fn message(failure: &Failure, lines: &[String], log: Option<&Path>) -> String {
    let mut text = sentence(failure);
    let last = &lines[lines.len().saturating_sub(TAIL_LINES)..];
    if !last.is_empty() {
        text.push_str("\n\nThe server's last lines:\n");
        text.push_str(&last.join("\n"));
    } else if matches!(failure, Failure::Stopped(_) | Failure::TimedOut) {
        text.push_str("\n\nThe server wrote nothing before that.");
    }
    if let Some(path) = log {
        text.push_str("\n\nThe server's log: ");
        text.push_str(&path.to_string_lossy());
    }
    text
}

pub fn keep(folder: &Path, lines: &[String]) -> Option<PathBuf> {
    if lines.is_empty() {
        return None;
    }
    std::fs::create_dir_all(folder).ok()?;
    let path = folder.join(LOG_NAME);
    let mut text = lines.join("\n");
    text.push('\n');
    std::fs::write(&path, text).ok()?;
    Some(path)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn the_tail_keeps_the_newest_lines_whole_and_short() {
        let mut tail = Tail::default();
        for n in 0..(KEPT_LINES + 5) {
            tail.push(&format!("line {n}"));
        }
        tail.push("   ");
        tail.push("\u{1b}[31mred\u{7}");
        tail.push(&"x".repeat(1000));
        let lines = tail.lines();
        assert_eq!(lines.len(), KEPT_LINES);
        assert_eq!(lines[0], "line 7");
        assert_eq!(lines[KEPT_LINES - 2], "[31mred");
        assert_eq!(lines[KEPT_LINES - 1].len(), LINE_CHARS);
    }

    #[test]
    fn every_failure_has_its_sentence_the_last_lines_and_the_log() {
        let log = Path::new("/logs/form-md-server.log");
        let said = vec!["form.md is already running on /data (engine process 7).".to_string()];
        let busy = message(&Failure::Stopped(Some(BUSY_CODE)), &said, Some(log));
        assert!(busy.starts_with("form.md could not start: another copy of its server holds its data folder."));
        assert!(busy.contains("The server's last lines:\nform.md is already running on /data (engine process 7)."));
        assert!(busy.ends_with("The server's log: /logs/form-md-server.log"));
        assert!(message(&Failure::Stopped(Some(1)), &[], Some(log)).contains("stopped with code 1 before the window could open."));
        assert!(message(&Failure::Stopped(None), &[], None).ends_with("The server wrote nothing before that."));
        assert_eq!(
            sentence(&Failure::NotStarted("Access is denied.".to_string())),
            "The form.md server could not start: Access is denied."
        );
        assert_eq!(message(&Failure::Window("no display".to_string()), &[], None), "form.md could not open its window: no display");
    }

    #[test]
    fn the_wait_for_the_address_is_ninety_seconds_and_says_so() {
        assert_eq!(ADDRESS_WAIT, Duration::from_secs(90));
        let text = message(&Failure::TimedOut, &[], None);
        assert!(text.contains("waited 90 seconds for its server to say which address it serves on"));
        assert!(text.ends_with("The server wrote nothing before that."));
    }

    #[test]
    fn the_dialog_shows_only_the_last_twelve_lines() {
        let lines: Vec<String> = (0..40).map(|n| format!("line {n}")).collect();
        let text = message(&Failure::Stopped(Some(1)), &lines, None);
        assert!(text.contains("The server's last lines:\nline 28\n"));
        assert!(text.ends_with("line 39"));
        assert!(!text.contains("line 27"));
    }

    #[test]
    fn what_the_server_wrote_is_kept_in_a_file_and_nothing_is_written_for_nothing() {
        let folder = std::env::temp_dir().join(format!("form-md-engine-start-{}", std::process::id())).join("logs");
        assert_eq!(keep(&folder, &[]), None);
        assert!(!folder.exists());
        let lines = vec!["one".to_string(), "two".to_string()];
        let path = keep(&folder, &lines).expect("the file is written");
        assert_eq!(path, folder.join("form-md-server.log"));
        assert_eq!(std::fs::read_to_string(&path).unwrap(), "one\ntwo\n");
        let _ = std::fs::remove_dir_all(folder.parent().unwrap());
    }
}
