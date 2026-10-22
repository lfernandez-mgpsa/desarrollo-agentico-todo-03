# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

Aplicación TODO en Go (módulo `todo`) con Buffalo v1.1 + Pop v6 sobre PostgreSQL, renderizado en servidor con plantillas Plush. Código, mensajes de UI, docs y commits están en español.

## Comandos

```bash
docker compose up -d db          # PostgreSQL 17 en localhost:5432 (postgres/postgres)
buffalo dev                      # recarga en caliente; compila ./cmd/app (ver .buffalo.dev.yml)
go run ./cmd/app                 # alternativa sin Buffalo CLI; aplica migraciones al arrancar
docker compose up --build -d     # app + db en contenedores; healthcheck en /healthz
```

Validación (lo mismo que ejecuta CI en `.github/workflows/ci.yaml`):

```bash
gofmt -l .                       # debe salir vacío
go vet ./...
GO_ENV=test soda migrate -e test # una vez: crea el esquema en todo_test (soda: github.com/gobuffalo/pop/v6/soda@v6.4.1)
GO_ENV=test go test -p 1 ./...
go build ./...
```

- `-p 1` es obligatorio: todos los paquetes de test comparten la base `todo_test`.
- La base `todo_test` debe existir (compose solo crea `todo_development`): `docker compose exec db createdb -U postgres todo_test`.
- Un test concreto: `GO_ENV=test go test -p 1 ./actions -run Test_ActionSuite -testify.m Test_TasksCreate_Valid` (los tests son métodos de suites testify; `-run` selecciona la suite y `-testify.m` el método).
- `models` abre la conexión en `init()` según `GO_ENV`, así que cualquier paquete que lo importe (incluido `actions`) necesita PostgreSQL accesible incluso para compilar-y-ejecutar tests.

## Arquitectura

- `cmd/app/main.go`: ejecuta `models.Migrate()` (migraciones embebidas con `go:embed` en `migrations/`) y luego sirve `actions.App()`. No hace falta `soda` en desarrollo/producción; sí para preparar la base de test.
- `actions/app.go`: única definición de rutas y middleware (forceSSL solo en producción, CSRF, `popmw.Transaction`). Cada request va en una transacción; los handlers la obtienen con `c.Value("tx").(*pop.Connection)`, nunca usan `models.DB` directamente.
- Patrón de handler (ver `actions/tasks.go`): `Find` → 404 con `c.Error`; `ValidateAndCreate/Update`; si hay `verrs` se re-renderiza la plantilla con status 422 y `errors`; si no, flash + redirect 303 a `/`.
- `models/task.go`: la validación vive en `Validate` (título obligatorio, máx. `TitleMaxLength` = 200 runas, coherente con `size: 200` de la migración). Los campos no editables llevan `form:"-"`; `TasksUpdate` bindea a un struct propio para limitar qué se actualiza.
- Templates (`templates/`) y assets (`public/`) se embeben en el binario (`embed.go` en cada carpeta); un archivo nuevo se incluye automáticamente. Layout: `application.plush.html`.
- Los formularios PUT/DELETE usan `_method` oculto + `authenticity_token` (CSRF). `public/assets/app.js` gestiona `data-confirm` e intercepta el formulario de toggle (`data-task-toggle`) para actualizar la UI en cliente; también gestiona el reorden por arrastre y teclado (`data-task-reorder`, `PUT /tasks/reorder`). Reglas para formularios y su JS en `docs/formularios-js.md`.
- Tests: `actions` usa `suite.NewActionWithFixtures` con escenarios TOML en `fixtures/` (`as.LoadFixture("tres tareas")`); `models` usa `ModelSuite`.
- Migraciones: `.fizz` para esquema, `.sql` para datos semilla; nombrado `YYYYMMDDHHMMSS_nombre.{up,down}.*`.

## Flujo agéntico (ADW)

- Skills en `.claude/skills/`: `/feature`, `/bug`, `/chore` solo investigan y escriben un plan en `plans/`; `/implement plans/<plan>.md` lo ejecuta sin rediseñar, sin commit/push/PR.
- `/adw <encargo | URL de issue> [--tipo feature|bug|chore] [--remote origin] [--upstream upstream|none] [--base main]` encadena todo: clasifica y elige `/feature`, `/bug` o `/chore`, crea la rama (`feat/`, `fix/`, `chore/`), ejecuta `/implement`, valida como CI, hace commit con `/git:commit`, publica la rama en el fork y abre la PR en el fork y otra en el original desde la rama del fork (borradores si no pudieron ejecutarse los tests; omite la del original si la rama arrastra commits que el original no tiene). No pregunta: ante un bloqueo se detiene y lo reporta.
- Comandos en `.claude/commands/`: `/prime*` (contexto), `/env:start` (levanta el entorno, `--docker` para todo en contenedores), `/git:commit` (Conventional Commits, primera línea ≤ 50 caracteres, `git add` con lista explícita).
- `adws/`: trigger en Go (`bash adws/trigger-github.sh [--once]` o `go run ./adws`) que consulta issues de GitHub con la etiqueta de `config/adw.yml` y lanza `claude -p` con el `prompt` configurado (por defecto `/adw {{issue_url}} …`, que invoca la skill `/adw`). `repo` viene vacío y debe configurarse. Sus tests (`go test -p 1 ./adws`) simulan `gh` y `claude`. Detalles en `adws/README.md`.
