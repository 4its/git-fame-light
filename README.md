# gitfame-light · Go CLI

![Go Version](https://img.shields.io/badge/Go-%3E=1.21-blue)
![License](https://img.shields.io/badge/license-MIT-green)
![Platform](https://img.shields.io/badge/platform-macOS%20%7C%20Linux%20%7C%20Windows-lightgrey)
![CI](https://img.shields.io/badge/CI-ready-success)

Минималистичная утилита на Go для подсчёта вкладов в репозитории Git: **коммиты**, **добавленные/удалённые строки** и **чистый вклад (net)**.  
Работает локально, без Python и без внешних зависимостей кроме [`go-git`](https://github.com/go-git/go-git).

> По духу — лёгкий аналог `git-fame`: считает по **истории коммитов** за заданный период и агрегирует по авторам.

---

## ✨ Возможности

- ⏱️ Подсчёт за **период** (`--since`, `--until`) — даты в `YYYY-MM-DD` или `RFC3339`.
- 👤 Фильтр по **автору** (подстрока имени/почты).
- 🔀 Исключение **merge-коммитов** по умолчанию (опция `--include-merges`).
- 📊 Сводная таблица по авторам: commits, added, deleted, net.
- 🧾 Экспорт в **CSV** (`--csv=out.csv`) — удобно для отчётов.
- 🧱 Без внешних бинарников: только Go-модуль.

---

## 🚀 Установка

```bash
# Клонируем и собираем
git clone https://github.com/4its/git-fame-light.git
cd gitfame-light
go build -o gitfame-light

# или сразу запустить без сборки
go run .
```
## 🧰 Быстрый старт

Подсчитать вклад по всем авторам в репозитории за период:
```bash
./gitfame-light \
  --repo=/path/to/repo \
  --since=2025-07-21 \
  --until=2025-08-21
```

Только для конкретного автора:
```bash
./gitfame-light \
  --repo=~/dev/finmart-be \
  --since=2025-07-21 \
  --until=2025-08-21 \
  --author="egiazaryan"
```

Сохранить в CSV:
```bash
./gitfame-light \
  --repo=~/dev/finmart-be \
  --since=2025-07-21 \
  --until=2025-08-21 \
  --csv=stats.csv
```

Включить merge-коммиты (по умолчанию исключены):
```bash
./gitfame-light \
  --repo=~/dev/finmart-be \
  --since=2025-07-21 \
  --until=2025-08-21 \
  --include-merges
```

## ⚙️ Опции CLI
| Флаг               | Тип    | По умолчанию | Описание                                                                 |
| ------------------ | ------ | ------------ | ------------------------------------------------------------------------ |
| `--repo`           | string | `.`          | Путь к локальному git-репозиторию                                        |
| `--since`          | string | `now-30d`    | Начало периода (формат `YYYY-MM-DD`, `YYYY-MM-DD HH:MM`, либо `RFC3339`) |
| `--until`          | string | `now`        | Конец периода (те же форматы; для `YYYY-MM-DD` берётся 23:59:59 локали)  |
| `--author`         | string | пусто        | Фильтр по подстроке имени/почты (case-insensitive)                       |
| `--include-merges` | bool   | `false`      | Включать merge-коммиты                                                   |
| `--csv`            | string | пусто        | Путь для сохранения CSV                                                  |

>Таймзона по умолчанию — Europe/Moscow (можно заменить в коде под свою).

## 🖨️ Пример вывода
```bash
Repo: /Users/george/dev/finmart-be
Period: 2025-07-21T00:00:00+02:00 .. 2025-08-21T23:59:59+02:00 (Europe/Moscow)
Merges: excluded

Author                       Commits    Added   Deleted       Net
------------------------------------------------------------------
George Egiazaryan                 24      914       352       562
Teammate A                        17      480       510       -30
(unknown)                          1        8         0         8
------------------------------------------------------------------
TOTAL                             42     1402       862       540
```
CSV содержит столбцы: `author_name,author_email,commits,added,deleted,net` + строку `TOTAL`.

## 🧪 Фильтрация по файлам (опционально)
По умолчанию учитываются все изменения. Если нужно считать только, например, `*.go` — добавь фильтрацию внутри цикла по `stats`:
```go
if !strings.HasSuffix(strings.ToLower(s.Name), ".go") { 
    continue 
}
```

## 📦 Многорепозиторный сценарий
```bash
BASE=~/work/my-group
for repo in "$BASE"/*/.git; do
  [ -d "$repo" ] || continue
  dir="$(dirname "$repo")"
  echo "== $dir =="
  ./gitfame-light --repo="$dir" --since=2025-07-21 --until=2025-08-21
done
```

## 🧠 Как это работает

* История коммитов берётся через go-git
* Для каждого коммита считаются Patch.Stats() → суммируются Addition/Deletion.
* Мерджи по умолчанию исключаются (часто «шум» для вкладов).

>Важно: метрика — это сумма изменений в диффах коммитов, а не текущее число строк в репозитории.

## 🚧 Ограничения
* Переименования считаются как удаление+добавление (git стандарт).
* Бинарные файлы не учитываются в строках.
* Очень большие истории могут считаться долго.

## 🗺️ Roadmap
1. [ ] Фильтр по маскам файлов (--include-glob, --exclude-glob)
2. [ ] Группировка по директориям/модулям
3. [ ] Вывод в JSON
4. [ ] Агрегация по дням/неделям
5. [ ] Обработка нескольких репозиториев параллельно

## 📝 Лицензия

MIT © 2025 — George Egiazaryan/4its
