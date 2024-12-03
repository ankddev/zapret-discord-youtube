use std::{env, fs, io, process::Command};

pub fn get_options() -> Vec<String> {
    let current_dir = env::current_dir().expect("Failed to get current directory");
    let sub_dir = current_dir.join("pre-configs");
    let mut options = vec![
        "Delete service from autorun".to_string(),
        "Run BLOCKCHECK (Auto-setting BAT parameters)".to_string(),
    ];

    if let Ok(read_dir) = fs::read_dir(&sub_dir) {
        let mut bat_files: Vec<String> = read_dir
            .filter_map(|entry| {
                entry.ok().and_then(|e| {
                    let path = e.path();
                    if path.extension().and_then(|ext| ext.to_str()) == Some("bat") {
                        path.file_name().and_then(|n| n.to_str()).map(String::from)
                    } else {
                        None
                    }
                })
            })
            .collect();

        bat_files.sort_by(|a, b| custom_sort(a, b));
        options.extend(bat_files);
    }

    options
}

fn custom_sort(a: &str, b: &str) -> std::cmp::Ordering {
    // Split filenames into components for hierarchical sorting
    let (a_base, a_variant, a_provider, a_has_provider) = split_filename(a);
    let (b_base, b_variant, b_provider, b_has_provider) = split_filename(b);

    // First compare by base name
    match a_base.cmp(&b_base) {
        std::cmp::Ordering::Equal => {
            // Same base name, compare variants
            match a_variant.cmp(&b_variant) {
                std::cmp::Ordering::Equal => {
                    // Same variant, non-provider version comes first
                    match (a_has_provider, b_has_provider) {
                        (false, true) => std::cmp::Ordering::Less,
                        (true, false) => std::cmp::Ordering::Greater,
                        // Both have or don't have providers, sort by provider name
                        _ => a_provider.cmp(&b_provider),
                    }
                }
                // Different variants
                other => other,
            }
        }
        // Different base names
        other => other,
    }
}

fn split_filename(name: &str) -> (String, String, String, bool) {
    let without_ext = name.trim_end_matches(".bat");

    // Split into base name and parentheses part
    let (main_part, parentheses) = match without_ext.find('(') {
        Some(idx) => (&without_ext[..idx - 1], &without_ext[idx..]),
        None => (without_ext, ""),
    };

    // Split main part into components by underscore
    let parts: Vec<&str> = main_part.split('_').collect();
    let base = parts[0].to_string();

    // Get variant part (ALT, v2, etc)
    let variant = parts.get(1..).map_or(String::new(), |p| p.join("_"));

    // Check if it's a provider variant
    let has_provider = !parentheses.is_empty();

    (base, variant, parentheses.to_string(), has_provider)
}

pub fn run_powershell_command_with_output(command: &str) -> io::Result<String> {
    println!("{}", command);
    let output = Command::new("powershell")
        .args(&["-NoProfile", "-NonInteractive", "-Command", command])
        .output()
        .map_err(|e| {
            io::Error::new(
                io::ErrorKind::Other,
                format!("Failed to execute command: {}", e),
            )
        })?;

    let stdout = String::from_utf8_lossy(&output.stdout).to_string();
    let stderr = String::from_utf8_lossy(&output.stderr).to_string();

    if output.status.success() {
        if !stderr.is_empty() {
            Err(io::Error::new(io::ErrorKind::Other, stderr))
        } else {
            Ok(stdout)
        }
    } else {
        let error_message = if !stderr.is_empty() {
            stderr
        } else if !stdout.is_empty() {
            stdout
        } else {
            "Unknown error occurred while executing PowerShell command".to_string()
        };

        Err(io::Error::new(io::ErrorKind::Other, error_message))
    }
}

pub fn run_powershell_command(command: &str) -> io::Result<()> {
    let output = Command::new("powershell")
        .args(&["-Command", command])
        .output()
        .map_err(|e| {
            io::Error::new(
                io::ErrorKind::Other,
                format!("Failed to execute command: {}", e),
            )
        })?;

    if output.status.success() {
        Ok(())
    } else {
        let error_message = String::from_utf8_lossy(&output.stderr).into_owned();

        let error_message = if error_message.is_empty() {
            String::from_utf8_lossy(&output.stdout).into_owned()
        } else {
            error_message
        };

        let error_message = if error_message.is_empty() {
            "Unknown error occured while executing PowerShell command".to_string()
        } else {
            error_message
        };

        Err(io::Error::new(io::ErrorKind::Other, error_message))
    }
}
