@echo off
chcp 65001 >nul
:: 65001 - UTF-8

cd /d "%~dp0..\"
set BIN=%~dp0..\bin\

set LIST_TITLE=ZAPRET: Ubisoft Fix
set LIST_PATH=%~dp0..\lists\list-ubisoft.txt
set UBISOFT_IPSET_PATH=%~dp0..\lists\ipset-ubisoft.txt

start "%LIST_TITLE%" /min "%BIN%winws.exe" ^
--wf-tcp=80,443 --wf-udp=443,3074 ^
--filter-tcp=80 --hostlist="%LIST_PATH%" --ipset="%UBISOFT_IPSET_PATH%" --dpi-desync=fake,split2 --dpi-desync-autottl=2 --dpi-desync-fooling=md5sig --new ^
--filter-tcp=443 --hostlist="%LIST_PATH%" --ipset="%UBISOFT_IPSET_PATH%" --dpi-desync=fake,split --dpi-desync-autottl=2 --dpi-desync-repeats=6 --dpi-desync-fooling=badseq --dpi-desync-fake-tls="%BIN%tls_clienthello_www_google_com.bin" --new ^
--filter-udp=443 --hostlist="%LIST_PATH%" --ipset="%UBISOFT_IPSET_PATH%" --dpi-desync=fake --dpi-desync-repeats=6 --dpi-desync-fake-quic="%BIN%quic_initial_www_google_com.bin" --new ^
--filter-udp=3074 --hostlist="%LIST_PATH%" --ipset="%UBISOFT_IPSET_PATH%" --dpi-desync=fake --dpi-desync-repeats=6 --dpi-desync-udplen-increment=10
