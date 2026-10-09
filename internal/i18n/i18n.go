package i18n

import (
	"fmt"
	"sort"
)

// Lang is a supported interface language.
type Lang string

const (
	EN Lang = "en"
	RU Lang = "ru"
	UK Lang = "uk"
	BE Lang = "be"
	DE Lang = "de"
)

// Names holds the autonym of each language, used in the language picker.
var Names = map[Lang]string{
	EN: "English",
	RU: "Русский",
	UK: "Українська",
	BE: "Беларуская",
	DE: "Deutsch",
}

// Order is the order the languages are offered in.
var Order = []Lang{EN, RU, UK, BE, DE}

var dict = map[Lang]map[string]string{
	RU: {
		"app.name":         "Android Device Management (TUI)",
		"app.tagline":      "Обёртка над adb и fastboot для Linux",
		"lang.title":       "Выберите язык интерфейса",
		"lang.hint":        "↑/↓ — выбор, enter — подтвердить, q — выход",
		"common.ok":        "ОК",
		"common.cancel":    "Отмена",
		"common.yes":       "Да",
		"common.no":        "Нет",
		"common.back":      "Назад",
		"common.quit":      "Выход",
		"common.exit":      "Завершение работы",
		"common.confirm":   "Подтверждение",
		"common.select":    "Выбор",
		"common.error":     "Ошибка",
		"common.run":       "Выполнить",
		"common.loading":   "Загрузка…",
		"common.working":   "Выполняется…",
		"common.done":      "Готово",
		"common.failed":    "Не удалось",
		"common.exitcode":  "Код возврата: %d",
		"common.none":      "нет",
		"common.enabled":   "Включён",
		"common.disabled":  "Отключён",
		"common.refresh":   "Обновить список устройств",
		"common.missing":   "отсутствует",
		"common.found":     "найдено",
		"common.topbottom": "вверх/вниз",
		"common.enter":     "Enter — выбрать, d — папка, enter на папке — войти",
		"common.nav":       "↑/↓ — навигация, влево/вправо или h/l — вверх/вниз, d — в папку, enter — выбрать, q — отмена",

		"pre.title":           "Проверка окружения",
		"pre.setup.title":     "Подготовка окружения",
		"pre.setup.done":      "adb и fastboot готовы к работе",
		"pre.setup.foot":      "Сейчас будет установлен сам интерфейс.",
		"pre.setup.ack":       "продолжить",
		"pre.setup.keepgoing": "Программа всё равно будет установлена, platform-tools можно поставить позже.",
		"pre.checking":        "Проверяю наличие adb и fastboot…",
		"pre.found":           "adb: %s · fastboot: %s",
		"pre.missing":         "Обнаружены отсутствующие компоненты: %s",
		"pre.question":        "Установить platform-tools (adb + fastboot) сейчас?",
		"pre.explain":         "Потребуется sudo и доступ к репозиториям вашей системы. Без этого программа не сможет работать с устройством.",
		"pre.distro":          "Обнаружена система: %s",
		"pre.pkg":             "Будут установлены пакеты: %s",
		"pre.cmd":             "Команда установки: %s",
		"pre.alt":             "Альтернативные варианты пакетов: %s",
		"pre.sudo":            "Откроется запрос пароля sudo прямо в терминале.",
		"pre.installing":      "Выполняю установку…",
		"pre.done":            "Установка завершена. Продолжаю.",
		"pre.failed":          "Установка не удалась. Проверьте вывод выше и запустите вручную.",
		"pre.aborted":         "Установка отменена.",
		"pre.nodetect":        "Не удалось определить дистрибутив. Введите команду установки вручную:",
		"pre.manual":          "Ручная установка",
		"pre.recheck":         "Повторная проверка окружения",
		"pre.sudo.fail":       "Не удалось получить права sudo. Установите вручную и перезапустите программу.",
		"pre.start.tip":       "Подключите устройство по USB и выберите его из списка.",
		"pre.udev.tip":        "Если устройство не определяется, проверьте udev-правила для Android.",

		"dev.title":            "Устройства",
		"dev.select":           "Выбор устройства для работы",
		"dev.pick":             "Выберите устройство, с которым будем работать",
		"dev.one":              "Обнаружено устройство",
		"dev.many":             "Обнаружено устройств: %d",
		"dev.none":             "Устройства не найдены",
		"dev.none.hint":        "Подключите телефон по USB, включите отладку по USB или переведите его в режим fastboot.",
		"dev.rescan":           "Найти устройства заново",
		"dev.serial":           "Серийный номер",
		"dev.codename":         "Кодовое имя",
		"dev.model":            "Модель",
		"dev.bootloader":       "Загрузчик",
		"dev.locked":           "ЗАБЛОКИРОВАН",
		"dev.unlocked":         "РАЗБЛОКИРОВАН",
		"dev.unknown":          "НЕИЗВЕСТНО",
		"dev.state":            "Состояние",
		"dev.state.offline":    "не в сети",
		"dev.state.unauth":     "не авторизован",
		"dev.state.bootloader": "режим fastboot",
		"dev.state.device":     "готов к работе",
		"dev.state.sideload":   "режим sideload",
		"dev.state.recovery":   "режим recovery",
		"dev.mode":             "Режим",
		"dev.via":              "Подключение",
		"dev.via.usb":          "USB",
		"dev.via.tcp":          "TCP",
		"dev.tcp.connect":      "Подключиться по TCP",
		"dev.tcp.disconn":      "Отключить TCP-устройство",
		"dev.tcp.host":         "Хост (например 192.168.1.10:5555)",
		"dev.sel.locked":       "Загрузчик заблокирован — доступен ограниченный набор команд.",
		"dev.sel.unlock":       "Загрузчик разблокирован — доступен полный функционал.",

		"menu.title":        "Главное меню",
		"menu.choose":       "Выберите раздел",
		"menu.device":       "Текущее устройство",
		"menu.cat.info":     "Информация об устройстве",
		"menu.cat.files":    "Файлы и хранилище",
		"menu.cat.apps":     "Приложения",
		"menu.cat.boot":     "Загрузка и перезагрузка",
		"menu.cat.flash":    "Прошивка (fastboot)",
		"menu.cat.recovery": "Recovery и sideload",
		"menu.cat.system":   "Система и оболочка",
		"menu.cat.network":  "Сеть и беспроводная отладка",
		"menu.cat.debug":    "Отладка и журналы",
		"menu.cat.other":    "Прочее",
		"menu.actions":      "Действия",
		"menu.switch":       "Сменить устройство",
		"menu.lang":         "Сменить язык",
		"menu.about":        "О программе",

		"cmd.title":                "Команды",
		"cmd.list":                 "%s — доступные команды",
		"cmd.locked.notice":        "Загрузчик заблокирован: показаны только заведомо рабочие команды.",
		"cmd.locked.unlock_all":    "Получить полный функционал (может не работать на заблокированном загрузчике)",
		"cmd.locked.back_safe":     "Вернуться к безопасному списку",
		"cmd.warn.title":           "Внимание",
		"cmd.warn.body":            "Эта команда может не сработать на заблокированном загрузчике и вернуть ошибку доступа. Продолжить?",
		"cmd.risk":                 "Рискованная операция",
		"cmd.risk.body":            "Команда изменяет разметку или данные устройства. Продолжить?",
		"cmd.args.title":           "Аргументы команды",
		"cmd.need.args":            "Введите необходимые аргументы",
		"cmd.need.pkg":             "Имя пакета",
		"cmd.need.part":            "Имя раздела (например boot, system, super)",
		"cmd.need.text":            "Текст",
		"cmd.need.file":            "Выберите файл",
		"cmd.need.dir":             "Выберите папку",
		"cmd.need.remote":          "Путь на устройстве (например /sdcard/Download)",
		"cmd.need.choose":          "Выберите вариант",
		"cmd.out":                  "Вывод",
		"cmd.save":                 "Сохранить вывод в файл",
		"cmd.saved":                "Вывод сохранён: %s",
		"cmd.save.fail":            "Не удалось сохранить: %v",
		"cmd.script.title":         "Скрипт не может быть запущен",
		"cmd.script.tool.adb":      "В скрипте нет ни одной команды adb, поэтому в режиме adb он не выполнится. Подключите телефон по USB с включённой отладкой либо выберите скрипт с командами adb.",
		"cmd.script.tool.fastboot": "В скрипте нет ни одной команды fastboot, поэтому в режиме fastboot он не выполнится. Переведите телефон в режим загрузчика либо выберите скрипт с командами fastboot.",
		"cmd.script.catfail":       "Не удалось прочитать скрипт: %v",
		"cmd.script.nofile":        "Файл скрипта не выбран.",
		"cmd.nothing":              "Нет доступных команд для этого режима.",

		"file.title":       "Выбор файла",
		"file.path":        "Путь",
		"file.pick.file":   "Выберите файл",
		"file.pick.dir":    "Выберите папку",
		"file.up":          "..  (вверх)",
		"file.hidden":      "Скрытые файлы",
		"file.selected":    "Выбрано: %s",
		"file.err.read":    "Не удалось прочитать папку: %v",
		"file.err.open":    "Не удалось открыть путь: %v",
		"file.err.perm":    "Нет прав доступа",
		"file.empty":       "Папка пуста",
		"file.size":        "%s",
		"file.enter.dir":   "Открыть папку",
		"file.select.here": "Выбрать текущий элемент",

		"about.author": "Автор: %s",
		"about.body":   "Обёртка над adb и fastboot.\nВерсия: %s\n\nУправление: ↑/↓ — навигация, enter — выбор, esc — назад, q — выход, / — поиск, ? — помощь.",
		"help.title":   "Управление",
		"help.body":    "↑/↓ или k/j — навигация\nenter — выбрать\nesc — назад\nq — выход\nd — открыть папку в браузере файлов\nh/l или ←/→ — вверх/вниз по дереву папок\nctrl+c — принудительный выход",

		"err.title":        "Ошибка",
		"err.adb":          "Не удалось получить данные adb: %v",
		"err.noser":        "Серийный номер устройства не определён.",
		"err.exec":         "Ошибка запуска: %v",
		"err.miss.arg":     "Не заполнен аргумент: %s",
		"err.quit":         "Программа завершена.",
		"err.device.left":  "Устройство отключилось.",
		"err.empty.output": "Команда завершена без вывода.",
	},
	EN: {
		"app.name":         "Android Device Management (TUI)",
		"app.tagline":      "A wrapper around adb and fastboot for Linux",
		"lang.title":       "Select interface language",
		"lang.hint":        "↑/↓ — choose, enter — confirm, q — quit",
		"common.ok":        "OK",
		"common.cancel":    "Cancel",
		"common.yes":       "Yes",
		"common.no":        "No",
		"common.back":      "Back",
		"common.quit":      "Quit",
		"common.exit":      "Exit",
		"common.confirm":   "Confirm",
		"common.select":    "Select",
		"common.error":     "Error",
		"common.run":       "Run",
		"common.loading":   "Loading…",
		"common.working":   "Working…",
		"common.done":      "Done",
		"common.failed":    "Failed",
		"common.exitcode":  "Exit code: %d",
		"common.none":      "none",
		"common.enabled":   "Enabled",
		"common.disabled":  "Disabled",
		"common.refresh":   "Refresh device list",
		"common.missing":   "missing",
		"common.found":     "found",
		"common.topbottom": "top/bottom",
		"common.enter":     "Enter — select, d — folder, enter on a folder — open",
		"common.nav":       "↑/↓ — navigate, ←/→ or h/l — up/down, d — open folder, enter — select, q — cancel",

		"pre.title":           "Environment check",
		"pre.setup.title":     "Environment setup",
		"pre.setup.done":      "adb and fastboot are ready",
		"pre.setup.foot":      "The interface itself is installed next.",
		"pre.setup.ack":       "continue",
		"pre.setup.keepgoing": "The program will still be installed; platform-tools can come later.",
		"pre.checking":        "Checking for adb and fastboot…",
		"pre.found":           "adb: %s · fastboot: %s",
		"pre.missing":         "Missing components detected: %s",
		"pre.question":        "Install platform-tools (adb + fastboot) now?",
		"pre.explain":         "sudo access and your distribution package repositories are required. Without them the program cannot talk to a device.",
		"pre.distro":          "Detected system: %s",
		"pre.pkg":             "Packages to install: %s",
		"pre.cmd":             "Install command: %s",
		"pre.alt":             "Alternative package names: %s",
		"pre.sudo":            "A sudo password prompt will appear in the terminal.",
		"pre.installing":      "Installing…",
		"pre.done":            "Installation finished. Continuing.",
		"pre.failed":          "Installation failed. Check the output above and run it manually.",
		"pre.aborted":         "Installation cancelled.",
		"pre.nodetect":        "Could not detect the distribution. Type the install command manually:",
		"pre.manual":          "Manual install",
		"pre.recheck":         "Re-check environment",
		"pre.sudo.fail":       "Could not obtain sudo privileges. Install manually and restart the program.",
		"pre.start.tip":       "Plug the device over USB and pick it from the list.",
		"pre.udev.tip":        "If the device is not detected, check the Android udev rules.",

		"dev.title":            "Devices",
		"dev.select":           "Select a device",
		"dev.pick":             "Choose the device to work with",
		"dev.one":              "1 device detected",
		"dev.many":             "Devices detected: %d",
		"dev.none":             "No devices found",
		"dev.none.hint":        "Plug the phone over USB, enable USB debugging or switch it to fastboot mode.",
		"dev.rescan":           "Scan devices again",
		"dev.serial":           "Serial",
		"dev.codename":         "Codename",
		"dev.model":            "Model",
		"dev.bootloader":       "Bootloader",
		"dev.locked":           "LOCKED",
		"dev.unlocked":         "UNLOCKED",
		"dev.unknown":          "UNKNOWN",
		"dev.state":            "State",
		"dev.state.offline":    "offline",
		"dev.state.unauth":     "unauthorized",
		"dev.state.bootloader": "fastboot mode",
		"dev.state.device":     "ready",
		"dev.state.sideload":   "sideload mode",
		"dev.state.recovery":   "recovery mode",
		"dev.mode":             "Mode",
		"dev.via":              "Transport",
		"dev.via.usb":          "USB",
		"dev.via.tcp":          "TCP",
		"dev.tcp.connect":      "Connect over TCP",
		"dev.tcp.disconn":      "Disconnect TCP device",
		"dev.tcp.host":         "Host (for example 192.168.1.10:5555)",
		"dev.sel.locked":       "Bootloader is locked — a limited set of commands is available.",
		"dev.sel.unlock":       "Bootloader is unlocked — full functionality is available.",

		"menu.title":        "Main menu",
		"menu.choose":       "Choose a section",
		"menu.device":       "Current device",
		"menu.cat.info":     "Device information",
		"menu.cat.files":    "Files and storage",
		"menu.cat.apps":     "Applications",
		"menu.cat.boot":     "Boot and reboot",
		"menu.cat.flash":    "Flashing (fastboot)",
		"menu.cat.recovery": "Recovery and sideload",
		"menu.cat.system":   "System and shell",
		"menu.cat.network":  "Network and wireless debugging",
		"menu.cat.debug":    "Debug and logs",
		"menu.cat.other":    "Miscellaneous",
		"menu.actions":      "Actions",
		"menu.switch":       "Switch device",
		"menu.lang":         "Change language",
		"menu.about":        "About",

		"cmd.title":                "Commands",
		"cmd.list":                 "%s — available commands",
		"cmd.locked.notice":        "Bootloader is locked: only commands expected to work are shown.",
		"cmd.locked.unlock_all":    "Get full functionality (may not work on a locked bootloader)",
		"cmd.locked.back_safe":     "Back to the safe list",
		"cmd.warn.title":           "Warning",
		"cmd.warn.body":            "This command may fail on a locked bootloader with an access error. Continue?",
		"cmd.risk":                 "Risky operation",
		"cmd.risk.body":            "The command changes the partition layout or device data. Continue?",
		"cmd.args.title":           "Command arguments",
		"cmd.need.args":            "Provide the required arguments",
		"cmd.need.pkg":             "Package name",
		"cmd.need.part":            "Partition name (for example boot, system, super)",
		"cmd.need.text":            "Text",
		"cmd.need.file":            "Pick a file",
		"cmd.need.dir":             "Pick a folder",
		"cmd.need.remote":          "Path on the device (for example /sdcard/Download)",
		"cmd.need.choose":          "Pick an option",
		"cmd.out":                  "Output",
		"cmd.save":                 "Save output to a file",
		"cmd.saved":                "Output saved: %s",
		"cmd.save.fail":            "Could not save: %v",
		"cmd.script.title":         "The script cannot be run",
		"cmd.script.tool.adb":      "The script contains no adb command, so it cannot run in adb mode. Connect the phone over USB with USB debugging on, or pick a script that uses adb.",
		"cmd.script.tool.fastboot": "The script contains no fastboot command, so it cannot run in fastboot mode. Put the phone into the bootloader, or pick a script that uses fastboot.",
		"cmd.script.catfail":       "Could not read the script: %v",
		"cmd.script.nofile":        "No script file was chosen.",
		"cmd.nothing":              "No commands available for this mode.",

		"file.title":       "File selection",
		"file.path":        "Path",
		"file.pick.file":   "Pick a file",
		"file.pick.dir":    "Pick a folder",
		"file.up":          "..  (up)",
		"file.hidden":      "Hidden files",
		"file.selected":    "Selected: %s",
		"file.err.read":    "Could not read the folder: %v",
		"file.err.open":    "Could not open the path: %v",
		"file.err.perm":    "Permission denied",
		"file.empty":       "Folder is empty",
		"file.size":        "%s",
		"file.enter.dir":   "Open folder",
		"file.select.here": "Select this entry",

		"about.author": "Author: %s",
		"about.body":   "A wrapper around adb and fastboot.\nVersion: %s\n\nKeys: ↑/↓ — navigate, enter — select, esc — back, q — quit, / — search, ? — help.",
		"help.title":   "Keyboard",
		"help.body":    "↑/↓ or k/j — navigate\nenter — select\nesc — back\nq — quit\nd — open folder in the file browser\nh/l or ←/→ — up/down the folder tree\nctrl+c — force quit",

		"err.title":        "Error",
		"err.adb":          "Could not get data from adb: %v",
		"err.noser":        "Device serial could not be determined.",
		"err.exec":         "Execution error: %v",
		"err.miss.arg":     "Argument is missing: %s",
		"err.quit":         "Program finished.",
		"err.device.left":  "The device disconnected.",
		"err.empty.output": "The command produced no output.",
	},
}

// missingKeys collects keys present in EN but absent in a translation, so the
// console can warn about gaps during development instead of silently showing
// English text.
func missingKeys(l Lang) []string {
	var out []string
	for k := range dict[EN] {
		if _, ok := dict[l][k]; !ok {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

// Missing reports untranslated keys for the given language.
func Missing(l Lang) []string { return missingKeys(l) }

// SetLanguage selects the active language. Unknown languages fall back to EN.
func SetLanguage(l Lang) {
	if _, ok := dict[l]; !ok {
		l = EN
	}
	cur = l
}

var cur = RU

// Get returns the translation of key in the active language.
func Get(key string) string {
	if m, ok := dict[cur]; ok {
		if s, ok := m[key]; ok {
			return s
		}
	}
	if s, ok := dict[EN][key]; ok {
		return s
	}
	return key
}

// Current returns the active language.
func Current() Lang { return cur }

// Has reports whether key is defined in English. It lets callers assert that
// every key they reference actually exists, instead of silently rendering the
// key itself when it does not.
func Has(key string) bool {
	_, ok := dict[EN][key]
	return ok
}

// Keys returns every key defined in English.
func Keys() []string {
	out := make([]string, 0, len(dict[EN]))
	for k := range dict[EN] {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// T is Get with formatting.
func T(key string, args ...any) string {
	return fmt.Sprintf(Get(key), args...)
}
