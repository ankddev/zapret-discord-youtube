use anyhow::Context;
use std::fs::File;
use std::io::{Read, Write};
use std::path::Path;
use walkdir::DirEntry;
use zip::write::SimpleFileOptions;
use zip::ZipWriter;

pub fn add_dir_to_zip(
    it: &mut dyn Iterator<Item = DirEntry>,
    prefix: &Path,
    writer: &mut ZipWriter<File>,
) -> anyhow::Result<()> {
    let prefix =
        dunce::canonicalize(Path::new(prefix).join("..")).expect("Failed to canonicalize folder");
    let options = SimpleFileOptions::default().unix_permissions(0o755);
    let mut buffer = Vec::new();
    for entry in it {
        let path = entry.path();
        let name = path.strip_prefix(&prefix)?;
        let path_as_string = name
            .to_str()
            .map(str::to_owned)
            .with_context(|| format!("{name:?} Is a Non UTF-8 Path"))?;

        if path.is_file() {
            writer.start_file(path_as_string, options)?;
            let mut f = File::open(path)?;

            f.read_to_end(&mut buffer)?;
            writer.write_all(&buffer)?;
            buffer.clear();
        } else if !name.as_os_str().is_empty() {
            // Only if not root! Avoids path spec / warning
            // and mapname conversion failed error on unzip
            writer.add_directory(path_as_string, options)?;
        }
    }
    Ok(())
}

pub fn add_file_to_zip(
    path: &Path,
    prefix: &Path,
    writer: &mut ZipWriter<File>,
) -> anyhow::Result<()> {
    let prefix = Path::new(prefix);
    let name = path.strip_prefix(prefix)?;
    let mut buffer = Vec::new();
    let options = SimpleFileOptions::default().unix_permissions(0o755);
    let path_as_string = name
        .to_str()
        .map(str::to_owned)
        .with_context(|| format!("{name:?} Is a Non UTF-8 Path"))?;
    writer.start_file(path_as_string, options)?;
    let mut f = File::open(path)?;

    f.read_to_end(&mut buffer)?;
    writer.write_all(&buffer)?;
    buffer.clear();
    Ok(())
}
