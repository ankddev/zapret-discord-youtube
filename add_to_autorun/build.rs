use {std::io, winresource::WindowsResource};

fn main() -> io::Result<()> {
    if cfg!(target_os = "windows") {
        let mut res = WindowsResource::new();
        res.set_icon("icon.ico")
            .set("InternalName", "Autorun BAT as Service by ANKDDEV");

        res.compile()?;
    }

    Ok(())
}
