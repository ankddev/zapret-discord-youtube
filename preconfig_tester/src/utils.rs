use std::{io, mem};
use std::mem::MaybeUninit;
use std::process::Command;
use std::thread::sleep;
use std::time::Duration;
use winapi::shared::minwindef::DWORD;
use winapi::um::processthreadsapi::{GetCurrentProcess, OpenProcessToken};
use winapi::um::securitybaseapi::GetTokenInformation;
use winapi::um::winnt::{TokenElevation, TOKEN_ELEVATION, TOKEN_QUERY};
use crate::TokenHandle;

pub fn is_elevated() -> bool {
    if !cfg!(target_os = "windows") {
        return false;
    }

    unsafe {
        let mut token = MaybeUninit::uninit();
        let status = OpenProcessToken(
            GetCurrentProcess(),
            TOKEN_QUERY,
            token.as_mut_ptr()
        );

        if status == 0 {
            return false;
        }

        let token = TokenHandle(token.assume_init());
        let mut elevation = TOKEN_ELEVATION { TokenIsElevated: 0 };
        let mut size: DWORD = 0;
        let elevation_ptr: *mut TOKEN_ELEVATION = &mut elevation;

        let status = GetTokenInformation(
            token.0,
            TokenElevation,
            elevation_ptr as *mut _,
            mem::size_of::<TOKEN_ELEVATION>() as DWORD,
            &mut size,
        );

        if status != 0 {
            elevation.TokenIsElevated != 0
        } else {
            false
        }
    }
}

pub fn request_elevation() -> io::Result<()> {
    let executable = std::env::current_exe()?;
    if let Some(executable) = executable.to_str() {
        let marker_path = std::env::temp_dir().join("bypass_checker_elevated.tmp");

        if marker_path.exists() {
            std::fs::remove_file(marker_path)?;
            return Err(io::Error::new(
                io::ErrorKind::PermissionDenied,
                "Failed to obtain admin privileges",
            ));
        }

        std::fs::write(&marker_path, "")?;

        let quoted_executable = format!("\"{}\"", executable);
        // Use start /b to prevent new console window creation and run PowerShell hidden
        let spawn_result = Command::new("cmd")
            .args(&[
                "/C",
                "start",
                "/b",
                "powershell",
                "-WindowStyle",
                "Hidden",
                "-Command",
                &format!(
                    "Start-Process -FilePath {} -ArgumentList '--elevated' -Verb RunAs",
                    quoted_executable
                ),
            ])
            .spawn();

        match &spawn_result {
            Ok(_) => {
                // Small delay to ensure the new process has started
                sleep(Duration::from_millis(300));
                Ok(())
            }
            Err(e) => {
                let _ = std::fs::remove_file(marker_path);
                Err(io::Error::new(e.kind(), e.to_string()))
            }
        }
    } else {
        Err(io::Error::new(
            io::ErrorKind::InvalidInput,
            "Invalid executable path",
        ))
    }
}
