@echo off
chcp 65001 >nul
:: 65001 - UTF-8

cd /d "%~dp0..\"
set BIN=%~dp0..\bin\

set LIST_TITLE=ZAPRET: Ultimate Fix ALT v11
set LIST_PATH=%~dp0..\lists\list-ultimate.txt
set DISCORD_IPSET_PATH=%~dp0..\lists\ipset-discord.txt

start "%LIST_TITLE%" /min "%BIN%winws.exe" ^
--wf-tcp=80,443 --wf-udp=443,50000-65535 ^
--filter-udp=443 --hostlist="%LIST_PATH%" --dpi-desync=fake,tamper --dpi-desync-repeats=9 --dpi-desync-udplen-increment=15 --dpi-desync-udplen-pattern=0xCAFEBABE --dpi-desync-fake-quic="%BIN%quic_initial_www_google_com.bin" --new ^
--filter-udp=50000-65535 --ipset="%DISCORD_IPSET_PATH%" --dpi-desync=fake,disorder2 --dpi-desync-any-protocol --dpi-desync-cutoff=d4 --dpi-desync-repeats=9 --new ^
--filter-tcp=80 --hostlist="%LIST_PATH%" --dpi-desync=fake,disorder2 --dpi-desync-autottl=4 --dpi-desync-fooling=md5sig --new ^
--filter-tcp=443 --hostlist="%LIST_PATH%" --dpi-desync=split,tamper --dpi-desync-split-pos=3 --dpi-desync-autottl=4 --dpi-desync-repeats=9 --dpi-desync-fooling=badseq --dpi-desync-fake-tls="%BIN%tls_clienthello_www_google_com.bin" 