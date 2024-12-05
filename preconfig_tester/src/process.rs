use crate::error::AppResult;
use std::path::Path;
use std::process::{Child, Command, Stdio};
use std::thread::sleep;
use std::time::{Duration, Instant};
use sysinfo::{ProcessRefreshKind, ProcessesToUpdate, RefreshKind, System};

pub struct ProcessManager {
    sys: System,
}

impl ProcessManager {
    pub fn new() -> Self {
        Self {
            sys: System::new_with_specifics(
                RefreshKind::new().with_processes(ProcessRefreshKind::everything()),
            ),
        }
    }

    pub fn run_batch_file(&self, batch_file: &Path) -> std::io::Result<Child> {
        Command::new("cmd")
            .args(["/C", &batch_file.to_string_lossy()])
            .stdout(Stdio::null())
            .stderr(Stdio::null())
            .spawn()
    }

    pub fn wait_for_process(&mut self, process_name: &str, timeout: Duration) -> bool {
        let start = Instant::now();

        while start.elapsed() < timeout {
            self.sys.refresh_processes(ProcessesToUpdate::All, true);

            if self
                .sys
                .processes()
                .values()
                .any(|process| process.name() == process_name)
            {
                return true;
            }

            sleep(Duration::from_millis(500));
        }

        false
    }

    pub fn cleanup_process(&mut self, child: &mut Child, process_name: &str) -> AppResult<()> {
        // First try to kill the child process
        let _ = child.kill();
        sleep(Duration::from_millis(500));

        // Then ensure the named process is terminated
        self.ensure_process_terminated(process_name);
        Ok(())
    }

    pub fn ensure_process_terminated(&mut self, process_name: &str) {
        for _ in 0..3 {
            // Try up to 3 times
            self.sys.refresh_processes(ProcessesToUpdate::All, true);
            let processes = self
                .sys
                .processes()
                .values()
                .filter(|process| process.name() == process_name)
                .collect::<Vec<_>>();

            if processes.is_empty() {
                return; // Process is gone, we're done
            }

            // Kill all instances of the process
            for process in processes {
                let _ = process.kill();
            }

            sleep(Duration::from_millis(200)); // Wait a bit before checking again
        }
    }

    fn kill_process_by_name(&mut self, process_name: &str) {
        self.ensure_process_terminated(process_name);
    }
}
