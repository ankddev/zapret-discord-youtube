@echo off
chcp 65001 >nul
:: 65001 - UTF-8

cd /d "%~dp0..\"
set BIN=%~dp0..\bin\

set LIST_TITLE=ZAPRET: Cloudflare Fix Rostelekom
set LIST_PATH=%~dp0..\lists\list-cloudflare.txt

start "%LIST_TITLE%" /min "%BIN%winws.exe" ^
--wf-tcp=80,443 --wf-udp=443 ^
--filter-tcp=80 --hostlist="%LIST_PATH%" --dpi-desync=fake,split2 --dpi-desync-autottl=3 --dpi-desync-fooling=md5sig --new ^
--filter-tcp=443 --hostlist="%LIST_PATH%" --dpi-desync=syndata,split2 --dpi-desync-split-pos=2 --dpi-desync-repeats=8 --dpi-desync-fooling=md5sig --dpi-desync-fake-tls="%BIN%tls_clienthello_www_google_com.bin" --new ^
--filter-udp=443 --hostlist="%LIST_PATH%" --dpi-desync=fake,split2 --dpi-desync-repeats=8 --dpi-desync-udplen-increment=15 --dpi-desync-fake-quic="%BIN%quic_initial_www_google_com.bin"
