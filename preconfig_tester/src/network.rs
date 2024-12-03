use crate::error::AppResult;
use std::time::Duration;
use ureq::{Agent, AgentBuilder, Error as UreqError, ErrorKind as UreqErrorKind, Transport};

pub struct NetworkChecker {
    agent: Agent,
    timeout: Duration,
}

impl NetworkChecker {
    pub fn new(timeout: Duration) -> Self {
        let agent = AgentBuilder::new()
            .timeout_read(timeout)
            .timeout_write(timeout)
            // Chrome 122 on Windows 10 User-Agent
            .user_agent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36")
            // Enable TLS session resumption
            .try_proxy_from_env(false)
            // Enable automatic redirect following (like browsers do)
            .redirects(4)
            .build();

        Self { agent, timeout }
    }

    pub fn test_connection(&self, target: &str) -> AppResult<bool> {
        // Add a small delay before testing to ensure previous connection is fully closed
        std::thread::sleep(Duration::from_millis(1000));

        let domain = target.split(':').next().unwrap_or(target);
        let result = self.try_connect(domain)?;

        // Add another delay after testing to ensure connection cleanup
        std::thread::sleep(Duration::from_millis(500));

        Ok(result == ConnectionResult::Success)
    }

    pub fn check_dpi_fingerprint(&self, domain: &str) -> AppResult<DPITestResult> {
        println!("Checking connection with {}...", domain);

        match self.try_connect(domain)? {
            ConnectionResult::Success => Ok(DPITestResult::NoDPI),
            ConnectionResult::ConnectionReset => Ok(DPITestResult::DPIDetected),
            ConnectionResult::NoConnection => Ok(DPITestResult::NoConnection),
            ConnectionResult::Timeout => Ok(DPITestResult::DPIDetected),
            ConnectionResult::ISPBlock => Ok(DPITestResult::ISPBlocked),
        }
    }

    fn try_connect(&self, domain: &str) -> AppResult<ConnectionResult> {
        let url = format!("https://{}", domain);
        println!("DEBUG: Trying to connect to {}...", url);

        // Create a new agent for each connection attempt to avoid connection reuse
        let agent = AgentBuilder::new()
            .timeout_read(self.timeout)
            .timeout_write(self.timeout)
            .user_agent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36")
            .try_proxy_from_env(false)
            .redirects(4)
            .build();

        // Build a request with browser-like headers
        let request = agent.get(&url)
            // Common headers
            .set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8")
            .set("Accept-Language", "en-US,en;q=0.9")
            .set("Accept-Encoding", "gzip, deflate, br")
            .set("Cache-Control", "max-age=0")
            .set("Connection", "keep-alive")
            .set("Upgrade-Insecure-Requests", "1")
            .set("DNT", "1")
            // Security headers
            .set("Sec-Fetch-Dest", "document")
            .set("Sec-Fetch-Mode", "navigate")
            .set("Sec-Fetch-Site", "none")
            .set("Sec-Fetch-User", "?1")
            // Chrome-specific headers
            .set("Sec-Ch-Ua", "\"Chromium\";v=\"122\", \"Not(A:Brand\";v=\"24\", \"Google Chrome\";v=\"122\"")
            .set("Sec-Ch-Ua-Mobile", "?0")
            .set("Sec-Ch-Ua-Platform", "\"Windows\"");

        match request.call() {
            Ok(response) => {
                // Check redirect
                let initial_url = url.to_lowercase();
                let final_url = response.get_url().to_lowercase();

                if initial_url != final_url {
                    println!("DEBUG: Redirected from {} to {}", initial_url, final_url);

                    let initial_domain = extract_domain(&initial_url);
                    let final_domain = extract_domain(&final_url);

                    if initial_domain != final_domain {
                        if is_block_page_url(&final_url) {
                            println!(
                                "DEBUG: Detected redirect to potential block page: {}",
                                final_url
                            );
                            return Ok(ConnectionResult::ISPBlock);
                        }
                    }
                }

                // Check response content
                match response.into_string() {
                    Ok(body) => {
                        if is_block_page_content(&body) {
                            println!("DEBUG: Detected block page content");
                            Ok(ConnectionResult::ISPBlock)
                        } else {
                            Ok(ConnectionResult::Success)
                        }
                    }
                    Err(e) => {
                        println!("DEBUG: Error reading response body: {}", e);
                        Ok(ConnectionResult::ConnectionReset)
                    }
                }
            }
            Err(UreqError::Status(code, response)) => {
                println!("DEBUG: HTTP error status: {}", code);

                if (300..400).contains(&code) {
                    if let Some(location) = response.header("location") {
                        if is_block_page_url(location) {
                            println!("DEBUG: Detected redirect to block page: {}", location);
                            return Ok(ConnectionResult::ISPBlock);
                        }
                    }
                }
                Ok(ConnectionResult::ConnectionReset)
            }
            Err(UreqError::Transport(transport)) => {
                println!("DEBUG: Transport error: {}", transport);
                handle_transport_error(&transport)
            }
        }
    }
}

fn handle_transport_error(transport: &Transport) -> AppResult<ConnectionResult> {
    match transport.kind() {
        UreqErrorKind::Io => {
            let err_string = transport.to_string().to_lowercase();
            if err_string.contains("timed out") || err_string.contains("timeout") {
                return Ok(ConnectionResult::Timeout);
            } else if err_string.contains("connection refused")
                || err_string.contains("connection reset")
                || err_string.contains("connection aborted")
            {
                return Ok(ConnectionResult::ConnectionReset);
            } else if err_string.contains("not found")
                || err_string.contains("name resolution failed")
                || err_string.contains("no such host")
            {
                return Ok(ConnectionResult::NoConnection);
            }
            println!("DEBUG: Other IO error: {}", transport);
            Ok(ConnectionResult::ConnectionReset)
        }
        UreqErrorKind::UnknownScheme | UreqErrorKind::InvalidUrl => {
            println!("DEBUG: Network error: {:?}", transport.kind());
            Ok(ConnectionResult::NoConnection)
        }
        _ => {
            println!("DEBUG: Other transport error: {:?}", transport.kind());
            Ok(ConnectionResult::ConnectionReset)
        }
    }
}

fn is_block_page_url(url: &str) -> bool {
    let url_lower = url.to_lowercase();
    url_lower.contains("block")
        || url_lower.contains("warning")
        || url_lower.contains("blocked")
        || url_lower.contains("rkn")
        || url_lower.contains("restriction")
}

fn is_block_page_content(body: &str) -> bool {
    let body_lower = body.to_lowercase();
    body_lower.contains("заблокирован")
        || body_lower.contains("доступ ограничен")
        || body_lower.contains("access denied")
        || body_lower.contains("blocked by")
        || body_lower.contains("webmaster@rkn.gov.ru")
}

fn extract_domain(url: &str) -> String {
    let without_protocol = url.split("://").nth(1).unwrap_or(url);
    without_protocol
        .split('/')
        .next()
        .unwrap_or(without_protocol)
        .to_string()
}

#[derive(Debug, PartialEq)]
enum ConnectionResult {
    Success,
    ConnectionReset,
    Timeout,
    NoConnection,
    ISPBlock,
}

#[derive(Debug, PartialEq)]
pub enum DPITestResult {
    NoDPI,
    DPIDetected,
    ISPBlocked,
    NoConnection,
    Unclear,
}

impl DPITestResult {
    pub fn to_english_string(&self) -> String {
        match self {
            DPITestResult::NoDPI => "DPI not found, site available directly".to_string(),
            DPITestResult::DPIDetected => "DPI locking found".to_string().to_uppercase(),
            DPITestResult::ISPBlocked => "Site locked by your internet provider"
                .to_string()
                .to_uppercase(),
            DPITestResult::NoConnection => "No connection with site".to_string(),
            DPITestResult::Unclear => "Check result is unclear".to_string(),
        }
    }
}
