<h1 align="center">zapret-discord-youtube</h1>
<h6 align="center">Сборка Zapret для Windows для исправления работы YouTube, Discord и Viber в России</h6>
<div align="center">
  <a href="https://github.com/ankddev/zapret-discord-youtube/releases"><img alt="GitHub Downloads" src="https://img.shields.io/github/downloads/ankddev/zapret-discord-youtube/total"></a>
  <a href="https://github.com/ankddev/zapret-discord-youtube/releases"><img alt="GitHub Release" src="https://img.shields.io/github/v/release/ankddev/zapret-discord-youtube"></a>
  <a href="https://github.com/ankddev/zapret-discord-youtube"><img alt="GitHub Repo stars" src="https://img.shields.io/github/stars/ankddev/zapret-discord-youtube?style=flat"></a>
</div>

Эта сбока включает в себя файлы из [оригинального репозитория](https://github.com/bol-van/zapret-win-bundle), кастомные пре-конфиги для исправления работы YouTube, Discord, Viber или других сервисов в России и некоторые полезные утилиты, написанные на Go.
# Настройка
## Скачать
Вы можете загрузить эту сборку из [релизов](https://github.com/ankddev/zapret-discord-youtube/releases) или [GitHub Actions](https://github.com/ankddev/zapret-discord-youtube/actions).
## Обновление
Вы можете обновить эту сборку, запустив `Check for updates.exe`. Он проверит наличие обновлений и скачает их, если они доступны.
## Использование
* Отключите все VPNы, Zapret, GoodbyeDPI, Warp и другой похожий софт
* **Разархивируйте** загуженный архив
* Перейдите в папку "pre-configs"
* Запустите один из батников в этой папке:
  * UltimateFix or GeneralFix - Discord, YouTube и выбранные домены
  * DiscordFix - Discord
  * YouTubeFix - YouTube
  * ViberFix - Viber
* Наслаждайтесь!

> [!TIP]
> Также вы можете запустить файл `Run pre-config.exe` и выбррать пре-конфиг для запуска

## Добавить в автозапуск
Чтобы добавить фикс в автозапуск, запустите файл `Add to autorun.exe` и выберите один из представленных батников. Чтобы удалить фикс из автозапуска, запустите этот файл и выберите `Delete service from autorun`.

## Настройка для других сайтов
Вы можете добавить свои домены в `list-ultimate.txt` или использовать для этого специальную утилиту. Запустите файл `Set domain list.exe` и выберите все домены, которые хотите, потом выберите `Save list` и нажмите <kbd>ENTER</kbd>.

Лист `russia-blacklist.txt` содержит все [известные заблокированные](https://antizapret.prostovpn.org/domains-export.txt) в России сайты.

# Устанение ошибок
## Ни один пре-конфиг не помог
Во-первых, проверьте **все** пре-конфиги или запустите `Automatically search pre-config.exe`. Если это не помогает, используйте BLOCKCHECK.

* Запустите `blockcheck.cmd`
* Введите домен для проверки
* Ip protocol version равна `4`
* Отметьте `HTTP`, `HTTPS 1.2`, `HTTPS 1.3` и `HTTP3 QUIC` (введите `Y` для этих пунктов)
* Verify certificates не надо отмечать (введите `N`)
* Retry test 1 или 2 раза (введите `1` или `2`)
* Connection mode равен `2`
* Подождите
* Вы увидите `* SUMMARY` и `press enter to continue`. Закройте это окно
* Откройте `blockcheck.log` в текстовом редакторе
* Найдите строку `* SUMMARY` (в конце файла)
* TЗдесь вы найдёте аргументы для winws, например, `winws --wf-l3=ipv4 --wf-tcp=80 --dpi-desync=split2 --dpi-desync-split-http-req=host`
* Также работающие статегии отмечены `!!!!! AVAILABLE !!!!!`
* Создайте файл `custom.bat` (или какой-то другой) и заполните, использую другие пре-конфиги как пример
* Запустите `custom.bat`

## Файл winws.exe не найден
Распакуйте архив перед запуском. Также, ваш антивирус мог удалить файлы, пожалуйста, отключите его или добавьте папку фикса в исключения.

## Не могу удалить файлы
* Остановите сервис и удалите его из автозапуска
* Закройте окно winws.exe
* Остановите и очистите WinDivert
* Удалите папку

## WinDivert не найден
* Проверьте, присутствует ли WinDivert
* Запустите в терминале:
```bash
sc stop windivert
sc delete windivert
sc stop windivert14
sc delete windivert14
```
* Запустите фикс заново

## Есть ли здесь вирусы?
В этой сборке нет вирусов, если вы скачали её с https://github.com/ankddev/zapret-discord-youtube/releases. Если ваш антивирус детектит вирусы, пожалуйста, добавьте папку с фиксом в исключения или отключите антивирус.
Здесь есть ответ разработчика оригинального репозитория по поводу детекций: https://github.com/bol-van/zapret/issues/393

## Как очистить WinDivert
Запустите в терминале:
```bash
sc stop WinDivert
sc delete WinDivert
```

## Как настроить Zapret на Linux
Check [this](https://github.com/bol-van/zapret/blob/master/docs/quick_start.txt).

# Внесение вклада
* Форкните репозиторий
* Клонируйте форк
* Создайте новую ветку
* Внесите изменения
* Запустите линтер и отформатируйте код:
```bash
go fmt .\...
```
* Создайте PR

## Сборка
Чтобы собрать этот проект, запустите:
```bash
scripts\build.bat
```
Это скомпилиррует все бинарные файлы и создаст zip архив в папке `build`.
## Структура проекта
Этот проект разделён на нескольео папок:
* `bin` содержит готовые бинарники из оригинального репозитория
* `pre-configs` содержит пре-конфиги (батники)
* `lists` содержит списки доменов
* `resources` содержит файл `blockcheck.cmd`
* `scripts` содержит скрипты для сборки проекта
* `cmd` содержит исходный код для утилит
  * `add_to_autorun` содержит код для утилиты, которая помогает добавить фикс в автозапуск
  * `select_domains` содержит код для утилиты, которая помогает выбрать домены для DPI
  * `preconfig_tester` помогает тестировать пре-конфиги
  * `run_preconfig` помогает запускать пре-конфиги
# Кредиты
* [Zapret](https://github.com/bol-van/zapret)
* [Zapret Win Bundle](https://github.com/bol-van/zapret-win-bundle)
* [WinDivert](https://github.com/basil00/WinDivert)
* [Zapret Discord](https://github.com/Flowseal/zapret-discord-youtube)
* [Zapret Discord YouTube](https://howdyho.net/windows-software/discord-fix-snova-rabotayushij-diskord-vojs-zvonki)
