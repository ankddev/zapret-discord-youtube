use crossterm::{
    cursor, execute,
    terminal::{self, Clear, ClearType},
};
use std::io::{self, Write};


/// Go to alternate screen, hide cursor and clear screen
pub fn init(stdout: &mut impl Write) -> io::Result<()> {
    terminal::enable_raw_mode()?;
    execute!(
        stdout,
        terminal::EnterAlternateScreen,
        cursor::Hide,
        Clear(ClearType::All),
        cursor::MoveTo(0, 0)
    )?;
    Ok(())
}


/// Get size of current terminal window
pub fn get_size() -> io::Result<(u16, u16)> {
    terminal::size()
}

/// Clean terminal screen
pub fn cleanup_terminal() -> io::Result<()> {
    let mut stdout = std::io::stdout();
    terminal::disable_raw_mode()?;
    execute!(
        stdout,
        Clear(ClearType::All),
        cursor::Show,
        terminal::LeaveAlternateScreen
    )?;
    stdout.flush()?;
    Ok(())
}

/// Setup clearing terminal screen on Ctrl-C
pub fn setup_terminal_cleanup() {
    let original_hook = std::panic::take_hook();
    std::panic::set_hook(Box::new(move |panic_info| {
        let _ = cleanup_terminal();
        original_hook(panic_info);
    }));

    ctrlc::set_handler(move || {
        let _ = cleanup_terminal();
        std::process::exit(0);
    })
    .expect("Error setting Ctrl-C handler");
}

/// Clean screen and exit
pub fn cleanup_and_exit(mut stdout: &mut impl Write) -> io::Result<()> {
    writeln!(&mut stdout, "\nReady!\nYou can close this window")
        .expect("Failed to print line to stdout");
    let mut input = String::new();
    io::stdin().read_line(&mut input)?;
    Ok(())
}
