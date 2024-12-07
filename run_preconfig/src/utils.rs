use std::{env, fs};

pub fn get_options() -> Vec<String> {
    let current_dir = env::current_dir().expect("Failed to get current directory");
    let sub_dir = current_dir.join("pre-configs");
    let mut options = vec![];

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
