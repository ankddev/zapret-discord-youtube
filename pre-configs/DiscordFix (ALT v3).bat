@echo off
chcp 65001 >nul
:: 65001 - UTF-8

cd /d "%~dp0..\"
set BIN=%~dp0..\bin\

set LIST_TITLE=ZAPRET: Discord Fix ALT v3
set LIST_PATH=%~dp0..\lists\list-discord.txt
set DISCORD_IPSET_PATH=%~dp0..\lists\ipset-discord.txt

start "%LIST_TITLE%" /min "%BIN%winws.exe" ^
--wf-tcp=443 --wf-udp=443,50000-65535 ^
--filter-tcp=80 --dpi-desync=fake,disorder2 --dpi-desync-autottl=4 --dpi-desync-fooling=md5sig --new ^
--filter-udp=443 --hostlist="%LIST_PATH%" --dpi-desync=fake,split2 --dpi-desync-cutoff=n4 --dpi-desync-udplen-increment=20 --dpi-desync-repeats=10 --dpi-desync-udplen-pattern=0xFEEDFACE --dpi-desync-fake-quic="%BIN%quic_initial_www_google_com.bin" --new ^
--filter-udp=50000-65535 --dpi-desync=fake,disorder2 --dpi-desync-any-protocol --dpi-desync-cutoff=d5 --dpi-desync-repeats=10 --dpi-desync-fake-quic="%BIN%quic_initial_www_google_com.bin" --new ^
--filter-tcp=443 --hostlist="%LIST_PATH%" --dpi-desync=split,tamper --dpi-desync-split-pos=3 --dpi-desync-autottl=4 --dpi-desync-repeats=10 --dpi-desync-fooling=badseq --dpi-desync-fake-tls="%BIN%tls_clienthello_www_google_com.bin" 