[🇬🇧 English](README.md) · [🇷🇺 Русский](README-ru.md)

# memorialiste

> La mémorialiste навещает ваш репозиторий, читает что изменилось с прошлого визита, пишет недостающие главы истории проекта и оставляет за собой merge request.

Однократно запускаемый CLI, который поддерживает документацию в актуальном
состоянии вслед за изменениями кода. Каждый прогон вычисляет `git diff` с
момента последнего обновления документации, вызывает OpenAI-совместимую LLM
для переписывания затронутых документов и открывает Merge/Pull Request.

## Как это работает (и зачем нужен манифест)

memorialiste **не** перегенерирует доки с нуля каждый раз. Вместо этого он
держит каждый doc-файл в синхроне с конкретным срезом исходников и
переделывает работу только когда этот срез изменился.

Связь между документами и кодом описана в одном файле:
**`docs/.docstructure.yaml`**. Он перечисляет все doc-файлы которыми
управляет тулза, и говорит ей какие пути исходников интересны каждому доку:

```yaml
docs:
  - path: docs/user/guide.md
    audience: end users
    covers: [cmd/, cliconfig/]
    description: Руководство пользователя по CLI.

  - path: docs/architecture.md
    audience: developers
    covers: [context/, generate/, output/, platform/]
    description: Внутренняя архитектура — пакеты, абстракции, потоки данных.
```

На каждом запуске для каждой записи memorialiste:

1. Читает watermark `generated_at` из frontmatter doc-файла (commit SHA на
   котором doc был последний раз сгенерирован).
2. Вычисляет git diff отфильтрованный только по путям `covers` этой записи.
3. Если diff пустой — **пропускает doc целиком**: ни одного LLM-вызова,
   токены не расходуются.
4. Иначе скармливает LLM diff + текущее тело doc'а, записывает обновлённое
   тело обратно с новым watermark'ом.

Поэтому манифест критичен: без per-doc `covers` каждый документ видел бы
каждое изменение, LLM тратила бы токены на отделение релевантного от
нерелевантного, а руководство пользователя переписывалось бы при каждой
правке во внутренних модулях.

Поле `audience` также используется для имени автоматически создаваемой
ветки (`docs/memorialiste-developers`, `docs/memorialiste-end-users`, …)
чтобы списки MR оставались читаемыми.

**Если манифест не найден** — memorialiste падает с понятной ошибкой и
exit code 1 ещё до любых обращений к LLM или git'у. Сообщение
подсказывает создать файл или указать другой путь через `--doc-structure`.

## Установка

```sh
docker pull idconstruct/memorialiste:latest
```

Закрепить конкретную версию для воспроизводимости:

```sh
docker pull idconstruct/memorialiste:v0.3.1
```

## Использование

### GitLab CI

```yaml
update-docs:
  image: idconstruct/memorialiste:latest
  variables:
    MEMORIALISTE_AST_CONTEXT: "true"
  script:
    - memorialiste
      --provider-url "$OLLAMA_URL"
      --model "qwen3-coder:30b"
      --platform gitlab
      --platform-token "$GITLAB_TOKEN"
      --project-id "$CI_PROJECT_ID"
      --dry-run=false
  rules:
    - if: $CI_COMMIT_BRANCH == "main"
```

### GitHub Actions

```yaml
- name: Update docs
  run: |
    docker run --rm --network=host \
      -v ${{ github.workspace }}:/repo \
      -e MEMORIALISTE_PLATFORM_TOKEN=${{ secrets.GITHUB_TOKEN }} \
      idconstruct/memorialiste:latest \
      --repo /repo \
      --provider-url "$OLLAMA_URL" \
      --model qwen3-coder:30b \
      --platform github \
      --project-id "${{ github.repository }}" \
      --dry-run=false \
      --ast-context
```

### Локальный dry-run

```sh
docker run --rm --network=host --user $(id -u):$(id -g) \
  -v $(pwd):/repo \
  idconstruct/memorialiste:latest \
  --repo /repo \
  --provider-url http://localhost:11434 \
  --model qwen3-coder:30b \
  --ast-context
```

## Claude, Gemini, GPT-4 и другие модели

memorialiste общается с LLM **исключительно по OpenAI-совместимому API
`/v1/chat/completions`**. Никаких нативных Anthropic / Google / OpenAI
SDK. Чтобы использовать не-Ollama модель, поднимите **OpenAI-совместимый
прокси**, переводящий запросы в нативное API целевого провайдера. Сам
memorialiste менять не нужно — достаточно указать в `--provider-url`
адрес прокси и поменять значение `--model`.

### Self-hosted: LiteLLM

[LiteLLM](https://github.com/BerriAI/litellm) поддерживает ~100 моделей
(Claude, Gemini, Bedrock, Vertex AI и т.д.) и запускается как Docker
сайдкар.

```yaml
# docker-compose.yml
services:
  litellm:
    image: ghcr.io/berriai/litellm:main-latest
    ports: ["4000:4000"]
    environment:
      ANTHROPIC_API_KEY: ${ANTHROPIC_API_KEY}
      OPENAI_API_KEY:    ${OPENAI_API_KEY}
```

```sh
memorialiste --provider-url http://litellm:4000 --model claude-3-5-sonnet-20241022
```

### Self-hosted: one-api

[one-api](https://github.com/songquanpeng/one-api) — агрегатор с
веб-интерфейсом, такой же OpenAI-совместимый API. Укажите его базовый
URL в `--provider-url`.

### Облако: OpenRouter

[OpenRouter](https://openrouter.ai) проксирует запросы к Claude, GPT-4,
Gemini и многим другим через единый OpenAI-совместимый endpoint.

```sh
memorialiste \
  --provider-url https://openrouter.ai/api/v1 \
  --api-key "$OPENROUTER_API_KEY" \
  --model anthropic/claude-3.5-sonnet
```

Значение `--api-key` отправляется провайдеру как
`Authorization: Bearer <key>`. Можно сочетать с `--model-params` для
настройки temperature, top_p и т.п.

## CLI-флаги и переменные окружения

Все флаги имеют env-аналог в формате uppercase snake_case с префиксом
`MEMORIALISTE_`. Флаг побеждает env при одновременном указании.

| Флаг | Env var | По умолчанию | Описание |
|------|---------|--------------|----------|
| `--provider-url` | `MEMORIALISTE_PROVIDER_URL` | `http://localhost:11434` | OpenAI-совместимый базовый URL |
| `--model` | `MEMORIALISTE_MODEL` | `qwen3-coder:30b` | Тег модели |
| `--model-params` | `MEMORIALISTE_MODEL_PARAMS` | `""` | JSON с дополнительными параметрами модели (`{"temperature":0.2}`) |
| `--system-prompt` | `MEMORIALISTE_SYSTEM_PROMPT` | встроенный | System prompt-литерал ИЛИ `@path/to/file` |
| `--prompt` | `MEMORIALISTE_PROMPT` | `""` | Дополнительный user prompt после diff'а |
| `--language` | `MEMORIALISTE_LANGUAGE` | `english` | Язык вывода; подставляется в плейсхолдер `{language}` |
| `--api-key` | `MEMORIALISTE_API_KEY` | `""` | Bearer-токен для LLM-провайдера |
| `--doc-structure` | `MEMORIALISTE_DOC_STRUCTURE` | `docs/.docstructure.yaml` | Путь к манифесту структуры документации |
| `--repo` | `MEMORIALISTE_REPO` | `.` | Корень локального git-репозитория |
| `--token-budget` | `MEMORIALISTE_TOKEN_BUDGET` | `12000` | Макс. токенов diff'а до запуска саммаризации |
| `--dry-run` | `MEMORIALISTE_DRY_RUN` | `true` | Писать файлы локально, не делать ветку/коммит/MR |
| `--branch-prefix` | `MEMORIALISTE_BRANCH_PREFIX` | `docs/memorialiste-` | Префикс имени ветки |
| `--ast-context` | `MEMORIALISTE_AST_CONTEXT` | `false` | AST-обогащённый diff через grep-ast |
| `--code-search` | `MEMORIALISTE_CODE_SEARCH` | `false` | Дать LLM tool `search_code` (function calling) |
| `--code-search-max-turns` | `MEMORIALISTE_CODE_SEARCH_MAX_TURNS` | `10` | Макс. число tool-call итераций до прерывания |
| `--repo-meta` | `MEMORIALISTE_REPO_META` | `basic` | Уровень метаданных репо: `basic` или `extended` |
| `--watermarks-file` | `MEMORIALISTE_WATERMARKS_FILE` | `""` | Внешний YAML-файл с watermark'ами; пусто — watermark живёт во frontmatter |
| `--platform` | `MEMORIALISTE_PLATFORM` | `gitlab` | `gitlab` или `github` |
| `--platform-url` | `MEMORIALISTE_PLATFORM_URL` | дефолт платформы | Базовый URL для self-hosted инстансов |
| `--platform-token` | `MEMORIALISTE_PLATFORM_TOKEN` | _обязателен (non-dry-run)_ | Токен доступа платформы |
| `--project-id` | `MEMORIALISTE_PROJECT_ID` | _обязателен (non-dry-run)_ | GitLab project ID или `owner/repo` для GitHub |
| `--base-branch` | `MEMORIALISTE_BASE_BRANCH` | `main` | Целевая ветка для открываемого MR/PR |
| `--version` | — | — | Вывести версию и выйти |
| `--help` | — | — | Сгруппированная помощь |

## Формат watermark

Каждый сгенерированный doc-файл несёт YAML frontmatter:

```markdown
---
generated_at: abc1234def5
---

# Заголовок документа
...
```

memorialiste читает `generated_at` чтобы посчитать diff с прошлого
запуска. Файл без frontmatter считается ни разу не сгенерированным —
берётся полная история по путям `covers` этой записи.

### Sidecar watermarks (чистый Markdown)

Чтобы держать сгенерированный Markdown без frontmatter, укажите
`watermarks_file` в манифесте (глобально в `defaults:` или per-entry):

```yaml
defaults:
  watermarks_file: .memorialiste-watermarks.yaml
docs:
  - path: docs/architecture.md
    covers: [internal/]
```

В sidecar-режиме файл документа пишется как есть, а SHA `generated_at`
для каждого файла лежит в отдельном YAML-файле:

```yaml
- path: docs/architecture.md
  generated_at: abc1234def5
```

Миграция между режимами двунаправленная и бесшовная за один переходный
прогон: если у документа есть frontmatter, но в sidecar-файле нет записи
— берётся frontmatter; если у документа нет frontmatter, но запись есть
в чужом sidecar-файле — берётся он. Следующая запись положит watermark
в каноничное место, указанное манифестом.

## Per-doc overrides

Записи (и опциональный блок `defaults:`) могут переопределить поля,
которые иначе берутся из CLI-флагов или env-переменных:

`model`, `model_params`, `language`, `system_prompt`, `prompt`,
`ast_context`, `code_search`, `code_search_max_turns`, `repo_meta`,
`token_budget`, `watermarks_file`.

Приоритет (по возрастанию): значения по умолчанию < блок `defaults` в
манифесте < per-doc запись манифеста < env-переменная
(`MEMORIALISTE_*`) < явно переданный CLI-флаг.

## Метаданные репозитория

LLM получает компактный блок метаданных, добавленный к промпту — чтобы
писать правильные номера версий:

```
=== Repository metadata ===
Latest tag: v0.3.1
HEAD: 53ebb4b...
Short SHA: 53ebb4b
=== End metadata ===
```

`--repo-meta=extended` добавляет remote URL (с замаскированными
токенами), ветку, последние 5 тегов с датами — полезно для CHANGELOG /
release-notes.

## AST-обогащённый контекст

`--ast-context` прогоняет каждый изменённый файл через TreeContext
рендерер grep-ast, поэтому модель видит сигнатуры окружающих функций и
структуру кода, а не только `+`/`-` строки. Существенно улучшает
качество для технических доков.

## AST Code Search

`--code-search` регистрирует у LLM функцию-обёртку `search_code` (через
механизм function calling). Прямо в процессе генерации модель может
запросить любую Go-декларацию в репозитории по regex-у на имени;
функция-обёртка возвращает тело функции/метода/типа с путём и диапазоном
строк. Полезно когда одного diff'а мало (например, документ одного
пакета упоминает символы из другого).

Ограничено `--code-search-max-turns` (по умолчанию 10) и per-file
таймаутом парсинга 5 секунд. Провайдер должен реализовывать
OpenAI-style function calling и возвращать настоящие `tool_calls`
(не строку с JSON в `content`). Проверено на локальной Ollama:
`qwen3:14b`, `qwen3.6:35b`, `gpt-oss:120b`. Модели возвращающие
`finish_reason: stop` с JSON в content (`qwen2.5-coder:7b`, иногда
`qwen3-coder:30b` на больших контекстах) — не следуют API; memorialiste
напечатает строку `WARNING — the model did not call any tools` и
продолжит без tool-результатов. Если провайдер вообще не принимает
запрос с tools — memorialiste падает с понятной ошибкой и
рекомендацией выставить `--code-search=false`.

**Совет — когда комбинировать с `--ast-context`**: AST-контекст уже
встраивает охватывающую функцию вокруг каждой изменённой строки, поэтому
tool-capable модели часто полностью игнорируют `search_code` если AST
включен. Используйте `--code-search` отдельно, когда хотите чтобы
модель доставала декларации **не упомянутые в diff'е**; используйте
оба флага вместе для максимального покрытия (модель сама выберет что
нужно).

## Диаграммы

Встроенный system prompt подсказывает модели вставлять Mermaid-диаграммы
(```` ```mermaid ```` блоки) при изменениях, затрагивающих архитектуру,
потоки данных или связи между компонентами. GitLab и GitHub нативно
рендерят Mermaid в превью Markdown. Никакого отдельного рендерящего
toolchain не требуется.

## Runtime-зависимости

Docker-образ включает:

| Tool | Версия | Зачем |
|------|--------|-------|
| `grep-ast` | 0.5.0 | AST-обогащённый diff (`--ast-context`) |
| `tree-sitter` | 0.20.4 | Зависимость grep-ast |
| `tree-sitter-languages` | 1.10.2 | Грамматики языков для grep-ast |

Используются только когда `--ast-context` включён.

## Примеры

Смотрите [`examples/`](examples/) — готовые сценарии:

| Сценарий | Что показывает |
|----------|----------------|
| [`01-user-guide`](examples/01-user-guide/) | Базовое руководство пользователя; встроенный prompt; минимум конфига |
| [`02-architecture`](examples/02-architecture/) | Архитектурный обзор для разработчиков с AST + Mermaid |
| [`03-developer-onboarding`](examples/03-developer-onboarding/) | Кастомный prompt для онбординга контрибьюторов |
| [`04-ai-readable`](examples/04-ai-readable/) | Плотный LLM-friendly контекст проекта (типа `CLAUDE.md`) |
| [`05-russian-docs`](examples/05-russian-docs/) | `--language russian` (работает для любого языка) |
| [`06-changelog`](examples/06-changelog/) | CHANGELOG через `--repo-meta=extended` |
| [`07-codesearch`](examples/07-codesearch/) | `--code-search` — модель сама подтягивает Go-декларации |
| [`ci-gitlab`](examples/ci-gitlab/) | Готовый `.gitlab-ci.yml` |
| [`ci-github`](examples/ci-github/) | Готовый GitHub Actions workflow |

В каждой папке сценария есть исполняемый `run.sh` — можно запускать
локально против работающей Ollama.

## Использование как Go-библиотека

memorialiste — также Go-библиотека. Можно использовать пакеты `manifest`,
`context`, `generate`, `output`, `platform` напрямую. См. godoc.

```go
import (
    "context"
    "github.com/inhuman/memorialiste/manifest"
    mctx "github.com/inhuman/memorialiste/context"
)

m, _ := manifest.Parse("docs/.docstructure.yaml")
dc, _ := mctx.Assemble(context.Background(), m.Docs[0], mctx.Options{
    RepoPath:    ".",
    ASTContext:  true,
    TokenBudget: 12000,
})
fmt.Println(dc.Diff)
```
