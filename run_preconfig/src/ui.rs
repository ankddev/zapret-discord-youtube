use crossterm::{
    cursor,
    event::{self, Event, KeyCode},
    execute,
    style::{Color, Print, ResetColor, SetForegroundColor},
    terminal::{Clear, ClearType},
};
use std::io::{self, Write};
use std::process::Command;
use std::time::{Duration, Instant};

#[derive(Debug)]
pub enum KeyAction {
    Exit,
    Select,
    None,
}

pub fn print_welcome_message() -> usize {
    let mut current_line = 0;
    println!("Welcome!");
    current_line += 1;
    println!("This program can run any pre-config (BAT file).");
    current_line += 1;
    println!("Author: ANKDDEV https://github.com/ankddev");
    current_line += 1;
    println!("Version: {}", env!("CARGO_PKG_VERSION"));
    current_line += 1;
    println!("===");
    current_line += 2;
    println!(
        "Using ARROWS on your keyboard, select BAT file from list for running it.\n"
    );
    println!("For selection press ENTER.");
    current_line += 2;
    current_line
}

pub fn render_options(
    stdout: &mut impl Write,
    options: &[String],
    current_selection: usize,
    start_row: usize,
    scroll_offset: usize,
    max_visible_options: usize,
) -> io::Result<()> {
    const MARKER: &str = "►";
    const EMPTY_MARKER: &str = " ";
    const SPACING: &str = " ";

    let visible_options = max_visible_options.saturating_sub(2);
    let total_options = options.len();

    let end_index = (scroll_offset + visible_options).min(total_options);
    let visible_range = scroll_offset..end_index;

    // Clear the options area only once
    execute!(
        stdout,
        cursor::MoveTo(0, start_row as u16),
        Clear(ClearType::FromCursorDown)
    )?;

    let mut current_row = start_row;

    // Build output buffer to minimize writes
    let mut output_buffer = Vec::new();

    // Up scroll indicator
    if scroll_offset > 0 {
        execute!(
            output_buffer,
            cursor::MoveTo(0, current_row as u16),
            SetForegroundColor(Color::DarkGrey),
            Print("↑ More options above"),
            ResetColor
        )?;
        current_row += 1;
    }

    // Display visible options
    for (index, option) in options.iter().enumerate() {
        if visible_range.contains(&index) {
            execute!(output_buffer, cursor::MoveTo(0, current_row as u16))?;

            if index == current_selection {
                execute!(
                    output_buffer,
                    SetForegroundColor(Color::Cyan),
                    Print(MARKER),
                    Print(SPACING),
                    Print(option),
                    ResetColor
                )?;
            } else {
                execute!(
                    output_buffer,
                    Print(EMPTY_MARKER),
                    Print(SPACING),
                    Print(option)
                )?;
            }

            current_row += 1;
        }
    }

    // Down scroll indicator
    if end_index < total_options {
        execute!(
            output_buffer,
            cursor::MoveTo(0, current_row as u16),
            SetForegroundColor(Color::DarkGrey),
            Print("↓ More options below"),
            ResetColor
        )?;
    }

    // Write the entire buffer at once
    stdout.write_all(&output_buffer)?;
    stdout.flush()?;

    Ok(())
}

pub fn handle_key_event(
    key: KeyCode,
    current_selection: &mut usize,
    scroll_offset: &mut usize,
    total_options: usize,
    max_visible_options: usize,
) -> Option<KeyAction> {
    let visible_options = max_visible_options.saturating_sub(2);

    match key {
        KeyCode::Up if *current_selection > 0 => {
            *current_selection -= 1;
            if *current_selection < *scroll_offset {
                *scroll_offset = *current_selection;
            }
            Some(KeyAction::None)
        }
        KeyCode::Down if *current_selection < total_options - 1 => {
            *current_selection += 1;
            if *current_selection >= *scroll_offset + visible_options {
                *scroll_offset = current_selection.saturating_sub(visible_options - 1);
            }
            Some(KeyAction::None)
        }
        KeyCode::PageUp => {
            *current_selection = current_selection.saturating_sub(visible_options);
            *scroll_offset = scroll_offset.saturating_sub(visible_options);
            Some(KeyAction::None)
        }
        KeyCode::PageDown => {
            *current_selection = (*current_selection + visible_options).min(total_options - 1);
            *scroll_offset = (*scroll_offset + visible_options)
                .min(total_options.saturating_sub(visible_options));
            Some(KeyAction::None)
        }
        KeyCode::Enter => Some(KeyAction::Select),
        KeyCode::Esc => Some(KeyAction::Exit),
        _ => None,
    }
}

pub fn handle_selection(
    stdout: &mut impl Write,
    options: &[String],
    current_selection: usize,
    message_row: usize,
) -> io::Result<()> {
    execute!(
        stdout,
        cursor::MoveTo(0, message_row as u16),
        Clear(ClearType::FromCursorDown)
    )?;
    stdout.flush()?;

    let selected_file = &options[current_selection][..];
    let current_dir = std::env::current_dir()?;
    let sub_dir = current_dir.join("pre-configs");
    let bat_file_path = sub_dir.join(selected_file);
    let output = Command::new(bat_file_path).output().map_err(|e| {
        io::Error::new(
            io::ErrorKind::Other,
            format!("Failed to execute command: {}", e),
        )
    }).expect("Failed to execute command");
    return if output.status.success() {
        Ok(())
    } else {
        let error_message = String::from_utf8_lossy(&output.stderr).into_owned();

        let error_message = if error_message.is_empty() {
            String::from_utf8_lossy(&output.stdout).into_owned()
        } else {
            error_message
        };

        let error_message = if error_message.is_empty() {
            "Unknown error occured while executing command".to_string()
        } else {
            error_message
        };

        Err(io::Error::new(io::ErrorKind::Other, error_message))
    };

    Ok(())
}

pub fn run_main_loop(
    stdout: &mut impl Write,
    options: &[String],
    start_row: usize,
    term_height: u16,
) -> io::Result<()> {
    let mut current_selection = 0;
    let mut scroll_offset = 0;
    let max_visible_options = std::cmp::min(15, term_height as usize - start_row - 3);

    let mut last_event_time = Instant::now();
    let mut last_render_time = Instant::now();

    // Define frame timing constants
    const FRAME_TIME: Duration = Duration::from_millis(33); // ~30 FPS
    const KEY_REPEAT_DELAY: Duration = Duration::from_millis(150);
    const EVENT_POLL_TIMEOUT: Duration = Duration::from_millis(16); // ~60 FPS polling

    // Initial render
    render_options(
        stdout,
        options,
        current_selection,
        start_row,
        scroll_offset,
        max_visible_options,
    )?;

    let mut needs_render = false;

    loop {
        let now = Instant::now();

        // Only poll for events if enough time has passed
        if event::poll(EVENT_POLL_TIMEOUT)? {
            if let Event::Key(event) = event::read()? {
                if now.duration_since(last_event_time) >= KEY_REPEAT_DELAY {
                    match handle_key_event(
                        event.code,
                        &mut current_selection,
                        &mut scroll_offset,
                        options.len(),
                        max_visible_options,
                    ) {
                        Some(KeyAction::Exit) => break,
                        Some(KeyAction::Select) => {
                            handle_selection(
                                stdout,
                                options,
                                current_selection,
                                start_row + max_visible_options + 1,
                            )?;
                            break;
                        }
                        Some(KeyAction::None) => {
                            needs_render = true;
                        }
                        None => {}
                    }
                    last_event_time = now;
                }
            }
        }

        // Only render if needed and enough time has passed since last render
        if needs_render && now.duration_since(last_render_time) >= FRAME_TIME {
            render_options(
                stdout,
                options,
                current_selection,
                start_row,
                scroll_offset,
                max_visible_options,
            )?;
            last_render_time = now;
            needs_render = false;
        }

        // Sleep for a small duration to prevent busy waiting
        if !needs_render {
            std::thread::sleep(Duration::from_millis(1));
        }
    }

    Ok(())
}
