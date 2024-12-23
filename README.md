<h1 align="center">zapret-discord-youtube</h1>
<h6 align="center">Zapret build for Windows for fixing YouTube, Discord, Viber in Russia</h6>
<div align="center">
  <a href="https://github.com/ankddev/zapret-discord-youtube/releases"><img alt="GitHub Downloads" src="https://img.shields.io/github/downloads/ankddev/zapret-discord-youtube/total"></a>
  <a href="https://github.com/ankddev/zapret-discord-youtube/releases"><img alt="GitHub Release" src="https://img.shields.io/github/v/release/ankddev/zapret-discord-youtube"></a>
  <a href="https://github.com/ankddev/zapret-discord-youtube"><img alt="GitHub Repo stars" src="https://img.shields.io/github/stars/ankddev/zapret-discord-youtube?style=flat"></a>
</div>

> [!WARNING]
> Currently zapret may not work for some users

[README на русском языке](./README.RU.md)

This build includes files from [original repository](https://github.com/bol-van/zapret-win-bundle), custom pre-configs for fixing YouTube, Discord, Viber or other services in Russia and some useful utilities, written in Go.
# Getting started
## Download
You can download this build from [releases](https://github.com/ankddev/zapret-discord-youtube/releases) or [GitHub Actions](https://github.com/ankddev/zapret-discord-youtube/actions).
## Updating
You can update this build by running `Check for updates.exe`. It will check for updates and download them if available.
## Usage
* Disable all VPNs, Zapret, GoodbyeDPI, Warp and other similar software
* **Unzip** downloaded archive
* Go to "pre-configs" folder
* Run one of BAT files in this folder
  * UltimateFix or GeneralFix - Discord, YouTube and selected domains
  * DiscordFix - Discord
  * YouTubeFix - YouTube
  * ViberFix - Viber
* Enjoy it!

> [!TIP]
> Also you can run file `Run pre-config.exe` and select pre-config to run

## Add to autorun
To add fix to autorun, start `Add to autorun.exe` and select one of presented BAT files. To delete from autorun, start this file and select `Delete service from autorun` option.

## Setup for other sites
You can add your own domains to `list-ultimate.txt` or you can use special utility for this. Start file `Set domain list.exe` and select all options you want, then select `Save list` and press <kbd>ENTER</kbd>.

List `russia-blacklist.txt` contains all [known blocked](https://antizapret.prostovpn.org/domains-export.txt) sites in Russia.

# Troubleshooting
## No one of pre-configs helps
Firstly, check **all** pre-configs or run `Automatically search pre-config.exe`. If this doesn't help you, use BLOCKCHECK.

* Run `blockcheck.cmd`
* Enter domain to check
* Ip protocol version is `4`
* Check `HTTP`, `HTTPS 1.2`, `HTTPS 1.3` and `HTTP3 QUIC` (enter `Y` for these entries)
* Not verify certificates (enter `N`)
* Retry test 1 or 2 times
* Connection mode is `2`
* Wait
* You will see `* SUMMARY` and `press enter to continue`. Close this window
* Open `blockcheck.log` in text editor
* Find `* SUMMARY` line in the end
* There you will find arguments to winws, for example `winws --wf-l3=ipv4 --wf-tcp=80 --dpi-desync=split2 --dpi-desync-split-http-req=host`
* Also working strategies marked with `!!!!! AVAILABLE !!!!!`
* Create file `custom.bat` (or anything else) and fill it using other pre-configs as example
* Run `custom.bat`

## File winws.exe not found
Unzip archive before starting. Also, your antivirus may block or delete it, please disable it or add fix folder to excluded folders.

## Can't delete files
* Stop service and delete from autorun
* Close winws.exe window
* Stop and clean WinDivert
* Delete folders

## WinDivert not found
* Check if WinDivert exists
* Run this:
```bash
sc stop windivert
sc delete windivert
sc stop windivert14
sc delete windivert14
```
* Run fix again

## Are there any viruses here
In this build there are no viruses, if you downloaded it from https://github.com/ankddev/zapret-discord-youtube/releases. If your antivirus detects viruses, please disable it or add fix folder to excluded folders.
Here is description about viruses: https://github.com/bol-van/zapret/issues/393

## How to clean WinDivert
Run this:
```bash
sc stop WinDivert
sc delete WinDivert
```

## How to setup Zapret on Linux
Check [this](https://github.com/bol-van/zapret/blob/master/docs/quick_start.txt).

# See Also

- [codewars-api-rs](https://github.com/ankddev/codewars-api-rs) - Rust library for Codewars API
- [conemu-progressbar-go](https://github.com/ankddev/conemu-progressbar-go) - Progress bar for ConEmu for Go
- [envfetch](https://github.com/ankddev/envfetch) - Lightweight crossplatform CLI tool for working with environment variables
- [terminal-go](https://github.com/ankddev/terminal-go) - Go library for working with ANSI/VT terminal sequences

# Contributing
* Fork this repo
* Clone your fork
* Create new branch
* Make changes
* Lint and format code
```bash
go fmt .\...
```
* Create pull request

## Building
To build this run this:
```bash
scripts\build.bat
```
This will generate binaries and zip archive in `build` folder.
## Structure of project
This project is separated in few folders:
* `bin` contains pre-built binaries from original repository
* `pre-configs` contains pre-configs (BAT files)
* `lists` contains lists of domains to work with
* `resources` contains `blockcheck.cmd` file
* `scripts` contains scripts for building and creating release archive
* `cmd` contains source code for utilities
  * `add_to_autorun` contains code for utility that helps you to add fix to autorun
  * `select_domains` contains source code for util that helps you to select domains for DPI
  * `preconfig_tester` helps you to test pre-configs
  * `run_preconfig` helps to run pre-configs
  * `check_for_updates` contains code for utility that checks if updates of fix available and downloads it
# Credits
* [Zapret](https://github.com/bol-van/zapret)
* [Zapret Win Bundle](https://github.com/bol-van/zapret-win-bundle)
* [WinDivert](https://github.com/basil00/WinDivert)
* [Zapret Discord](https://github.com/Flowseal/zapret-discord-youtube)
* [Zapret Discord YouTube](https://howdyho.net/windows-software/discord-fix-snova-rabotayushij-diskord-vojs-zvonki)
