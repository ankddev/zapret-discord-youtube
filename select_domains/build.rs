use std::io;
use winresource::WindowsResource;

fn main() -> io::Result<()> {
    if cfg!(target_os = "windows") {
        let mut res = WindowsResource::new();
        res.set_icon("icon.ico")
            .set("InternalName", "Select domains for Zapret DPI by ANKDDEV");

        res.compile()?;
    }

    Ok(())
}
