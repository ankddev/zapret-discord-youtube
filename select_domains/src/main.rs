use crossterm::{
    cursor::{self, Hide, Show},
    event::{self, Event, KeyCode, KeyEvent, KeyModifiers},
    execute, queue,
    style::Stylize,
    terminal::{self, ClearType, EnterAlternateScreen, LeaveAlternateScreen},
};
use std::fs::{self, File};
use std::io::{self, Read, Write};
use std::path::Path;
use std::thread;
use std::time::Duration;

#[derive(Debug)]
struct FileEntry {
    name: String,
    selected: bool,
    is_control: bool,
}

const VISIBLE_ITEMS: usize = 15;
const HEADER_LINES: usize = 5; // Header text + empty line + control buttons + empty line + separator
const SCROLL_AREA_HEIGHT: usize = VISIBLE_ITEMS + 2; // +2 for scroll indicators

fn main() -> io::Result<()> {
    terminal::enable_raw_mode()?;
    let mut stdout = io::stdout();
    execute!(stdout, EnterAlternateScreen, Hide)?;

    let result = run_app(&mut stdout);

    execute!(stdout, Show, LeaveAlternateScreen)?;
    terminal::disable_raw_mode()?;

    result
}

fn draw_screen(
    stdout: &mut io::Stdout,
    entries: &[FileEntry],
    current_index: usize,
    scroll_offset: usize,
    clear_screen: bool,
) -> io::Result<()> {
    if clear_screen {
        queue!(stdout, terminal::Clear(ClearType::All))?;
    }

    queue!(stdout, cursor::MoveTo(0, 0))?;

    // Header section
    writeln!(
        stdout,
        "Use ↑↓ arrows for navigation, SPACE or ENTER to select\n"
    )?;

    // Draw control options first
    let control_entries: Vec<_> = entries
        .iter()
        .enumerate()
        .filter(|(_, entry)| entry.is_control)
        .collect();

    for (index, entry) in &control_entries {
        let name = if entry.name == "SAVE LIST" {
            "SAVE LIST"
        } else {
            "CANCEL"
        };

        let line = format!(
            "{}  {}",
            if *index == current_index { ">" } else { " " },
            name
        );

        if *index == current_index {
            writeln!(stdout, "{}", line.reverse())?;
        } else {
            writeln!(stdout, "{}", line)?;
        }
    }

    writeln!(stdout)?; // Extra empty line after control options
    writeln!(stdout)?; // Separator line

    // Clear the scroll area
    for _ in 0..SCROLL_AREA_HEIGHT {
        writeln!(stdout, "{}", " ".repeat(50))?; // Clear line with spaces
    }

    // Move back to start of scroll area
    queue!(stdout, cursor::MoveTo(0, HEADER_LINES as u16))?;

    // Get file entries (non-control entries)
    let file_entries: Vec<_> = entries
        .iter()
        .enumerate()
        .filter(|(_, entry)| !entry.is_control)
        .collect();

    let total_files = file_entries.len();
    let visible_end = scroll_offset.saturating_add(VISIBLE_ITEMS).min(total_files);

    // Show scroll indicator if needed
    if scroll_offset > 0 {
        writeln!(stdout, " ↑ Scroll up for more files")?;
    } else {
        writeln!(stdout)?; // Keep spacing consistent
    }

    // Draw visible file entries
    let visible_entries = &file_entries[scroll_offset..visible_end];
    for entry in visible_entries {
        let real_index = entries.iter().position(|e| e.name == entry.1.name).unwrap();
        let line = format!(
            "{} {} {}",
            if real_index == current_index {
                ">"
            } else {
                " "
            },
            if entry.1.selected { "[+]" } else { "[ ]" },
            entry.1.name
        );

        if real_index == current_index {
            writeln!(stdout, "{}", line.reverse())?;
        } else {
            writeln!(stdout, "{}", line)?;
        }
    }

    // Move to the bottom scroll indicator position
    queue!(
        stdout,
        cursor::MoveTo(0, (HEADER_LINES + VISIBLE_ITEMS + 1) as u16)
    )?;

    // Show scroll indicator if needed
    if visible_end < total_files {
        writeln!(stdout, " ↓ Scroll down for more files")?;
    }

    stdout.flush()
}

fn join_selected_files(lists_dir: &Path, selected_entries: &[&FileEntry]) -> io::Result<()> {
    let ultimate_path = lists_dir.join("list-ultimate.txt");
    let mut ultimate_file = File::create(ultimate_path)?;

    for entry in selected_entries {
        let file_path = lists_dir.join(&entry.name);
        if file_path.exists() {
            let mut content = String::new();
            File::open(&file_path)?.read_to_string(&mut content)?;

            // Write the content
            writeln!(ultimate_file, "{}", content.trim())?;
        }
    }

    Ok(())
}

fn run_app(stdout: &mut io::Stdout) -> io::Result<()> {
    let lists_dir = Path::new("lists");
    if !lists_dir.exists() {
        fs::create_dir(lists_dir)?;
    }

    let config_path = lists_dir.join("selected.txt");
    let mut selected_files = Vec::new();
    if config_path.exists() {
        let mut content = String::new();
        File::open(&config_path)?.read_to_string(&mut content)?;
        selected_files = content.lines().map(String::from).collect();
    }

    // Create control entries first
    let mut entries = vec![
        FileEntry {
            name: String::from("SAVE LIST"),
            selected: false,
            is_control: true,
        },
        FileEntry {
            name: String::from("CANCEL"),
            selected: false,
            is_control: true,
        },
    ];

    // Add file entries
    let mut file_entries: Vec<FileEntry> = fs::read_dir(lists_dir)?
        .filter_map(|entry| {
            let entry = entry.ok()?;
            let name = entry.file_name().into_string().ok()?;
            if name.starts_with("list-") && name.ends_with(".txt") && name != "list-ultimate.txt" {
                Some(FileEntry {
                    name: name.clone(),
                    selected: selected_files.contains(&name),
                    is_control: false,
                })
            } else {
                None
            }
        })
        .collect();

    file_entries.sort_by(|a, b| a.name.cmp(&b.name));
    entries.extend(file_entries);

    let mut current_index = 0;
    let mut scroll_offset = 0;
    let num_control_entries = entries.iter().filter(|e| e.is_control).count();

    draw_screen(stdout, &entries, current_index, scroll_offset, true)?;

    'main: loop {
        if let Ok(true) = event::poll(Duration::from_millis(16)) {
            if let Ok(Event::Key(key)) = event::read() {
                let mut redraw = true;

                match key {
                    KeyEvent {
                        code: KeyCode::Up,
                        kind: event::KeyEventKind::Press,
                        ..
                    } => {
                        if current_index > 0 {
                            current_index -= 1;
                            if current_index >= num_control_entries {
                                let file_index = current_index - num_control_entries;
                                if scroll_offset > file_index {
                                    scroll_offset = file_index;
                                }
                            }
                        }
                    }
                    KeyEvent {
                        code: KeyCode::Down,
                        kind: event::KeyEventKind::Press,
                        ..
                    } => {
                        if current_index < entries.len() - 1 {
                            current_index += 1;
                            if current_index >= num_control_entries {
                                let file_index = current_index - num_control_entries;
                                if file_index >= scroll_offset + VISIBLE_ITEMS {
                                    scroll_offset = file_index - VISIBLE_ITEMS + 1;
                                }
                            }
                        }
                    }
                    KeyEvent {
                        code: KeyCode::Char(' ') | KeyCode::Enter,
                        kind: event::KeyEventKind::Press,
                        ..
                    } => {
                        match entries[current_index].name.as_str() {
                            "SAVE LIST" => {
                                // Save selected files to config
                                let mut file = File::create(&config_path)?;
                                let selected_entries: Vec<_> = entries
                                    .iter()
                                    .filter(|e| e.selected && !e.is_control)
                                    .collect();

                                for entry in &selected_entries {
                                    writeln!(file, "{}", entry.name)?;
                                }

                                // Join selected files into list-ultimate.txt
                                if let Err(e) = join_selected_files(lists_dir, &selected_entries) {
                                    execute!(
                                        stdout,
                                        cursor::MoveToNextLine(1),
                                        terminal::Clear(ClearType::FromCursorDown)
                                    )?;
                                    println!("{}", format!("Error occurred while merging files: {}. Exiting in 5 seconds...", e).red());
                                } else {
                                    execute!(
                                        stdout,
                                        cursor::MoveToNextLine(1),
                                        terminal::Clear(ClearType::FromCursorDown)
                                    )?;
                                    println!("{}", "Successful! List saved and files merged. Exiting in 5 seconds...".green());
                                }

                                stdout.flush()?;
                                thread::sleep(Duration::from_secs(5));
                                break 'main Ok(());
                            }
                            "CANCEL" => break 'main Ok(()),
                            _ => {
                                entries[current_index].selected = !entries[current_index].selected;
                            }
                        }
                    }
                    KeyEvent {
                        code: KeyCode::Char('c'),
                        modifiers: KeyModifiers::CONTROL,
                        kind: event::KeyEventKind::Press,
                        ..
                    } => {
                        break 'main Ok(());
                    }
                    _ => {
                        redraw = false;
                    }
                }

                if redraw {
                    draw_screen(stdout, &entries, current_index, scroll_offset, false)?;
                }
            }
        }
    }
}
