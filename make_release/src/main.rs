mod utils;

use spinners::{Spinner, Spinners};
use std::io::stdin;
use std::path::Path;
use std::{fs, io};
use walkdir::WalkDir;
use zip::write::ZipWriter;

fn main() -> io::Result<()> {
    print!("\x1b]9;4;3;\x1b\\");
    println!("[1/4] Preparing");

    let temp_dir = std::env::temp_dir();

    let current_directory = std::env::current_dir().expect("Failed to get current directory");
    assert!(current_directory.exists());

    let project_path = dunce::canonicalize(Path::new(env!("CARGO_MANIFEST_DIR")).join(".."))?;
    assert!(project_path.exists());

    let bin_path = project_path.join("bin");
    assert!(bin_path.exists());

    let lists_path = project_path.join("lists");
    assert!(lists_path.exists());

    let pre_configs_path = project_path.join("pre-configs");
    assert!(pre_configs_path.exists());

    let resources_path = project_path.join("resources");
    assert!(resources_path.exists());

    let add_to_autorun_path = current_directory.join("add_to_autorun.exe");
    assert!(add_to_autorun_path.exists());

    let select_domains_path = current_directory.join("select_domains.exe");
    assert!(select_domains_path.exists());

    let pre_config_tester_path = current_directory.join("preconfig_tester.exe");
    assert!(pre_config_tester_path.exists());

    let mut spinner = Spinner::new(Spinners::Dots, "[2/4] Copying files".into());
    let copy_options = fs_extra::file::CopyOptions::new().overwrite(true);

    let new_select_domains_path = temp_dir.join("Set domain list.exe");
    let new_pre_config_tester_path = temp_dir.join("Automatically search pre-config.exe");
    let new_add_to_autorun_path = temp_dir.join("Add to autorun.exe");
    let new_readme_path = temp_dir.join("___README.TXT");
    let new_blockcheck_path = temp_dir.join("blockcheck.cmd");

    fs_extra::file::copy(
        &select_domains_path,
        &new_select_domains_path,
        &copy_options,
    )
    .expect("Failed to copy `select_domains.exe`");
    fs_extra::file::copy(
        &pre_config_tester_path,
        &new_pre_config_tester_path,
        &copy_options,
    )
    .expect("Failed to copy `preconfig_tester.exe`");
    fs_extra::file::copy(
        &add_to_autorun_path,
        &new_add_to_autorun_path,
        &copy_options,
    )
    .expect("Failed to copy `add_to_autorun.exe`");
    fs_extra::file::copy(
        resources_path.join("___README.TXT"),
        &new_readme_path,
        &copy_options,
    )
    .expect("Failed to copy `___readme.txt`");
    fs_extra::file::copy(
        resources_path.join("blockcheck.cmd"),
        &new_blockcheck_path,
        &copy_options,
    )
    .expect("Failed to copy `blockcheck.cmd`");

    spinner.stop_and_persist("[2/4]", "Files copied".into());

    let mut spinner = Spinner::new(Spinners::Dots, "[3/4] Archiving files".into());

    let zip_path = current_directory.join("zapret-discord-youtube-ankddev.zip");
    let zip_file = fs::File::create(&zip_path).expect("Failed to create archive");
    let mut zip = ZipWriter::new(zip_file);

    // Bin folder
    let walkdir = WalkDir::new(&bin_path);
    let mut iterator = walkdir.into_iter().filter_map(|e| e.ok());
    utils::add_dir_to_zip(&mut iterator, &bin_path, &mut zip)
        .expect("Failed to add directory to zip");

    // Lists folder
    let walkdir = WalkDir::new(&lists_path);
    let mut iterator = walkdir.into_iter().filter_map(|e| e.ok());
    utils::add_dir_to_zip(&mut iterator, &lists_path, &mut zip)
        .expect("Failed to add directory to zip");

    // Pre-configs folder
    let walkdir = WalkDir::new(&pre_configs_path);
    let mut iterator = walkdir.into_iter().filter_map(|e| e.ok());
    utils::add_dir_to_zip(&mut iterator, &pre_configs_path, &mut zip)
        .expect("Failed to add directory to zip");

    utils::add_file_to_zip(&new_add_to_autorun_path, &temp_dir, &mut zip)
        .expect("Failed to add file to zip");
    utils::add_file_to_zip(&new_readme_path, &temp_dir, &mut zip)
        .expect("Failed to add file to zip");
    utils::add_file_to_zip(&new_select_domains_path, &temp_dir, &mut zip)
        .expect("Failed to add file to zip");
    utils::add_file_to_zip(&new_blockcheck_path, &temp_dir, &mut zip)
        .expect("Failed to add file to zip");
    utils::add_file_to_zip(&new_pre_config_tester_path, &temp_dir, &mut zip)
        .expect("Failed to add file to zip");

    zip.finish()?;

    spinner.stop_and_persist("[3/4]", "Files archived".into());

    let mut spinner = Spinner::new(Spinners::Dots, "[4/4] Cleaning cache".into());

    fs_extra::file::remove(&new_add_to_autorun_path).expect("Failed to delete file");
    fs_extra::file::remove(&new_readme_path).expect("Failed to delete file");
    fs_extra::file::remove(&new_select_domains_path).expect("Failed to delete file");
    fs_extra::file::remove(&new_blockcheck_path).expect("Failed to delete file");
    fs_extra::file::remove(&new_pre_config_tester_path).expect("Failed to delete file");

    spinner.stop_and_persist("[4/4]", "Cache cleaned".into());
    print!("\x1b]9;4;0;\x1b\\");

    println!(
        "Release build ready! Check `{}`.\nPress ENTER to continue.",
        &zip_path.display()
    );

    let mut input = String::new();
    stdin().read_line(&mut input).expect("Failed to read input");

    Ok(())
}
