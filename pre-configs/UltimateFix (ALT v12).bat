@echo off
chcp 65001 >nul
:: 65001 - UTF-8

cd /d "%~dp0..\"
set BIN=%~dp0..\bin\

set LIST_TITLE=ZAPRET: Ultimate Fix ALT v12
set LIST_PATH=%~dp0..\lists\list-ultimate.txt
set DISCORD_IPSET_PATH=%~dp0..\lists\ipset-discord.txt

start "%LIST_TITLE%" /min "%BIN%winws.exe" ^
--wf-tcp=80,443 --wf-udp=443,50000-65535 ^
--filter-udp=443 --hostlist="%LIST_PATH%" --dpi-desync=fake,split2 --dpi-desync-repeats=7 --dpi-desync-udplen-increment=20 --dpi-desync-udplen-pattern=0xFEEDFACE --dpi-desync-fake-quic="%BIN%quic_initial_www_google_com.bin" --new ^
--filter-udp=50000-65535 --ipset="%DISCORD_IPSET_PATH%" --dpi-desync=fake,split --dpi-desync-any-protocol --dpi-desync-cutoff=n3 --dpi-desync-repeats=7 --new ^
--filter-tcp=80 --hostlist="%LIST_PATH%" --dpi-desync=fake,tamper --dpi-desync-autottl=3 --dpi-desync-fooling=badseq --new ^
--filter-tcp=443 --hostlist="%LIST_PATH%" --dpi-desync=split2,disorder2 --dpi-desync-split-pos=2 --dpi-desync-autottl=3 --dpi-desync-repeats=7 --dpi-desync-fooling=md5sig --dpi-desync-fake-tls="%BIN%tls_clienthello_www_google_com.bin" 