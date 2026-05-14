# Examples

Готовые сценарии запуска memorialiste для разных типов документации. Каждый
пример самодостаточен — содержит свой `docstructure.yaml`, опциональный
системный prompt и shell-скрипт для локального запуска против Ollama.

## Когда что использовать

| Сценарий | Папка | Чем интересен |
|----------|-------|---------------|
| Руководство пользователя | [`01-user-guide/`](01-user-guide/) | Базовый минимум — встроенный prompt, без AST, dry-run |
| Архитектура для разработчиков | [`02-architecture/`](02-architecture/) | `--ast-context` для рендеринга функций; широкий `covers` |
| Онбординг контрибьюторов | [`03-developer-onboarding/`](03-developer-onboarding/) | Кастомный prompt через `@prompt.md`, фокус на "как работать с кодом" |
| LLM-friendly справка | [`04-ai-readable/`](04-ai-readable/) | Structured prose без декоративного форматирования, для feedback в чат-моделей |
| Документация на русском | [`05-russian-docs/`](05-russian-docs/) | `--language russian`, ориентация на не-английскую аудиторию |
| CHANGELOG из тегов | [`06-changelog/`](06-changelog/) | `--repo-meta=extended` для истории релизов, кастомный prompt |
| GitLab CI шаблон | [`ci-gitlab/`](ci-gitlab/) | `.gitlab-ci.yml` для автогенерации в pipeline |
| GitHub Actions шаблон | [`ci-github/`](ci-github/) | `docs.yml` workflow с `MEMORIALISTE_*` env vars |

## Как запустить любой из примеров локально

Каждая папка с doc-сценарием содержит исполняемый `run.sh`. Предполагается:

- запущенная локально Ollama на `http://localhost:11434`
- модель `qwen3-coder:30b` (или измените `MODEL` в `run.sh`)
- репозиторий memorialiste склонирован в текущей директории

```sh
cd examples/01-user-guide/
./run.sh
```

Скрипты используют `--dry-run=true` (по умолчанию) — файлы пишутся локально,
никаких коммитов и MR/PR. Перепроверь результат в `docs/...` и решай что
делать.

## Адаптация под свой репозиторий

1. Скопируй `docstructure.yaml` нужного примера в `docs/.docstructure.yaml`
   своего проекта.
2. Поправь `covers:` пути под структуру своего кода.
3. Поправь `path:` цели для генерируемых файлов.
4. Запусти `memorialiste --doc-structure docs/.docstructure.yaml ...`

## Дополнительная информация

- Полный список флагов и env vars — в [главном README](../README.md#cli-flags--environment-variables).
- Архитектура pipeline — [`docs/architecture.md`](../docs/architecture.md).
- Манифест-формат — [главный README, секция "Doc Structure Manifest"](../README.md#doc-structure-manifest).
