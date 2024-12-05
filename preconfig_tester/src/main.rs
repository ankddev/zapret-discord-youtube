use std::io::{self, stdin, stdout, Write};
use std::path::PathBuf;
use std::thread::sleep;
use std::time::Duration;

mod config;
mod domains;
mod error;
mod network;
mod process;
mod utils;

use crate::network::DPITestResult;
use config::Config;
use error::{AppError, AppResult};
use network::NetworkChecker;
use process::ProcessManager;

use winapi::um::handleapi::CloseHandle;

use termcolor::{Color, ColorChoice, ColorSpec, StandardStream, WriteColor};

struct TokenHandle(*mut winapi::ctypes::c_void);

impl Drop for TokenHandle {
    fn drop(&mut self) {
        if !self.0.is_null() {
            unsafe { CloseHandle(self.0) };
        }
    }
}

const DOMAIN_LIST: &[(&str, &str)] = &[
    ("1", "discord.com"),
    ("2", "youtube.com"),
    ("3", "spotify.com"),
    ("4", "speedtest.net"),
    ("5", "steampowered.com"),
    ("6", "custom"),
    ("0", "exit"),
];

const ORANGE: Color = Color::Rgb(252, 197, 108);
const GREEN: Color = Color::Rgb(126, 176, 0);
const BLUE: Color = Color::Rgb(87, 170, 247);
const MAGENTA: Color = Color::Rgb(196, 124, 186);
const RED: Color = Color::Rgb(214, 77, 91);

fn get_domain_choice() -> io::Result<String> {
    println!("\nSelect domain for checking:");
    for (number, domain) in DOMAIN_LIST {
        if *domain == "exit" {
            println!("{}. Exit", number);
        } else if *domain == "custom" {
            println!("{}. Enter your own domain", number);
        } else {
            println!("{}. {}", number, domain);
        }
    }

    loop {
        print!("\nEnter number of variant: ");
        stdout().flush()?;

        let mut choice = String::new();
        stdin().read_line(&mut choice)?;
        let choice = choice.trim();

        if let Some((_, domain)) = DOMAIN_LIST.iter().find(|(num, _)| *num == choice) {
            if *domain == "exit" {
                print!("Exiting..");
                stdout().flush()?;
                std::process::exit(0);
            } else if *domain == "custom" {
                print!("Enter domain (for example, example.com): ");
                stdout().flush()?;

                let mut custom_domain = String::new();
                stdin().read_line(&mut custom_domain)?;
                let custom_domain = custom_domain.trim().to_string();

                if domains::is_valid_domain(&custom_domain) {
                    return Ok(domains::format_domain_with_port(&custom_domain));
                } else {
                    println!("Invalid domain format. Use format domain.com");
                    continue;
                }
            } else {
                return Ok(domains::format_domain_with_port(domain));
            }
        } else {
            println!(
                "Invalid selection. Please select number from 0 to {}",
                DOMAIN_LIST.len() - 1
            );
        }
    }
}

fn main() -> AppResult<()> {
    let args: Vec<String> = std::env::args().collect();
    let is_elevated_instance = args.contains(&"--elevated".to_string());

    if is_elevated_instance {
        let marker_path = std::env::temp_dir().join("bypass_checker_elevated.tmp");
        if marker_path.exists() {
            let _ = std::fs::remove_file(marker_path);
        }
    }

    if !utils::is_elevated() {
        println!("Administrative privileges required for correct work of program.");
        println!("Please, confirm prompt for administrative privileges.");
        match utils::request_elevation() {
            Ok(_) => {
                // Exit immediately to close the non-elevated console window
                std::process::exit(0);
            }
            Err(e) => {
                eprintln!(
                    "Error occurred while requesting administrative privileges: {}",
                    e
                );
                let marker_path = std::env::temp_dir().join("bypass_checker_elevated.tmp");
                if marker_path.exists() {
                    let _ = std::fs::remove_file(marker_path);
                }
                // Give user time to read the error message
                sleep(Duration::from_secs(3));
                return Ok(());
            }
        }
    }

    let target_domain = get_domain_choice()
        .map_err(|e| AppError::IoError(format!("Error occurred while reading input: {}", e)))?;

    let config = Config {
        batch_dir: PathBuf::from("pre-configs"),
        target_domain,
        process_name: String::from("winws.exe"),
        process_wait_timeout: Duration::from_secs(10),
        connection_timeout: Duration::from_secs(5),
    };

    let process_manager = ProcessManager::new();
    let network_checker = NetworkChecker::new(config.connection_timeout);

    let result = run_bypass_check(config, process_manager, &network_checker);

    println!("\nPress Enter to exit...");
    stdout().flush().expect("Failed to flush buffer of output");
    let mut input = String::new();
    stdin().read_line(&mut input).expect("Failed to read input");

    result
}

fn run_bypass_check(
    config: Config,
    mut process_manager: ProcessManager,
    network_checker: &NetworkChecker,
) -> AppResult<()> {
    let mut stdout = StandardStream::stdout(ColorChoice::Always);

    let batch_files = config.get_batch_files()?;
    let mut success = false;

    let domain_without_port = config.target_domain.split(':').next().unwrap_or_default();

    println!("\nStarting testing domain: {}", config.target_domain);
    println!("------------------------------------------------");

    println!("Checking DPI blocks...");
    match network_checker.check_dpi_fingerprint(domain_without_port) {
        Ok(result) => {
            println!("Checking result: {}", result.to_english_string());

            if result == DPITestResult::NoDPI {
                println!("Using DPI spoofer not required.");
                return Ok(());
            }

            if result == DPITestResult::NoConnection {
                println!("Check internet connection and if domain is correct.");
                return Ok(());
            }

            println!("------------------------------------------------");
            println!("Testing pre-configs...");
        }
        Err(e) => {
            println!("Error occurred while checking: {}", e);
            println!("Testing pre-configs...");
        }
    }

    for batch_file in batch_files {
        stdout.set_color(ColorSpec::new().set_fg(Some(MAGENTA)))?;
        println!("\nRunning pre-config: {}", batch_file.display());
        stdout.set_color(ColorSpec::new().set_fg(Some(Color::White)))?;

        // before testing new batch file, ensure there's no winws.exe running in background
        process_manager.ensure_process_terminated(&config.process_name);

        let mut child = match process_manager.run_batch_file(&batch_file) {
            Ok(child) => child,
            Err(e) => {
                eprintln!("Failed to run pre-config {}: {}", batch_file.display(), e);
                continue;
            }
        };

        let process_result =
            process_manager.wait_for_process(&config.process_name, config.process_wait_timeout);

        if !process_result {
            eprintln!(
                "{} not started for pre-config {}",
                config.process_name,
                batch_file.display()
            );
            process_manager.cleanup_process(&mut child, &config.process_name)?;
            continue;
        }

        if network_checker.test_connection(&config.target_domain)? {
            let filename = batch_file
                .file_name()
                .and_then(|name| name.to_str())
                .unwrap_or("unknown");

            stdout.set_color(ColorSpec::new().set_fg(Some(GREEN)))?;
            println!("{}", format!("\n!!!!!!!!!!!!!\n[SUCCESS] It seems, this pre-config is suitable for you - {}\n!!!!!!!!!!!!!\n", filename));
            process_manager.cleanup_process(&mut child, &config.process_name)?;
            stdout.set_color(ColorSpec::new().set_fg(Some(Color::White)))?;
            success = true;
        } else {
            stdout.set_color(ColorSpec::new().set_fg(Some(RED)))?;
            println!(
                "{}",
                format!(
                    "[FAIL] Failed to establish connection using pre-config: {}",
                    batch_file.display()
                )
            );
            process_manager.cleanup_process(&mut child, &config.process_name)?;
            stdout.set_color(ColorSpec::new().set_fg(Some(Color::White)))?;
            continue;
        }
    }

    // Always try to clean up the process at the end
    process_manager.ensure_process_terminated(&config.process_name);

    // Double-check after a short delay
    sleep(Duration::from_millis(500));
    process_manager.ensure_process_terminated(&config.process_name);

    // If none of the pre-configs worked
    if !success {
        println!("\n------------------------------------------------");
        println!("Unfortunately, not found pre-config we can establish connection with :(");
        println!("Try to run BLOCKCHECK, to find necessary parameters for BAT file.");
    }

    Ok(())
}
