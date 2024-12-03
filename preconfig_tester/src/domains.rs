const DEFAULT_PORT: u16 = 443;

pub fn is_valid_domain(domain: &str) -> bool {
    if domain.is_empty() || domain.len() > 255 {
        return false;
    }

    !domain.contains(':')
        && domain
            .chars()
            .all(|c| c.is_ascii_alphanumeric() || c == '.' || c == '-')
        && !domain.starts_with('-')
        && !domain.ends_with('-')
}

pub fn format_domain_with_port(domain: &str) -> String {
    format!("{}:{}", domain.trim(), DEFAULT_PORT)
}
