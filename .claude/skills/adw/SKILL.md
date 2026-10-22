---
name: adw
description: Workflow completo de un encargo: clasifica, planifica con /feature, /bug o /chore, implementa con /implement, valida, hace commit con /git:commit, publica la rama y abre la PR en el fork y en el repositorio original
allowed-tools: Bash, Read, Glob, Grep, Write, Edit, Skill, Agent
model: claude-opus-5-5
---

# Workflow de desarrollo agéntico

`$ARGUMENTS` recibe lo mismo que `/feature`, `/bug` o `/chore` (la descripción del encargo) o bien la URL de una issue de GitHub, seguido opcionalmente de:

- `--tipo feature|bug|chore`: fuerza la clasificación en vez de decidirla.
- `--remote <nombre>`: remoto propio (fork) al que se publica la rama y donde se abre la primera PR (por defecto `origin`).
- `--upstream <nombre>`: remoto del repositorio original donde se abre la segunda PR, desde la rama del fork (por defecto `upstream`; `--upstream none` la omite).
- `--base <rama>`: rama base de la nueva rama y de ambas PR (por defecto `main`).

Ejemplos: `/adw el título de una tarea no puede superar 200 caracteres`, `/adw https://github.com/OWNER/REPO/issues/12 --remote origin --base main`.

Este workflow puede ejecutarse sin supervisión (`claude -p` desde `adws/`). No hagas preguntas: aplica las reglas de este documento y, si algo no puede resolverse con seguridad, detente en la fase correspondiente y explica el bloqueo. Nunca hagas push al remoto `--upstream` (solo recibe la PR), `push --force`, merge, cambios en `main` ni commits que no salgan de `/git:commit`.

## Modelos por fase

| Fase | Modelo | Cómo |
| --- | --- | --- |
| 1-3, 6-8 (preparar, clasificar, rama, comprobar, commit, PR) | Opus 5.5 | este workflow (`model: claude-opus-5-5`) |
| 4. Planificar | Opus 5.5 | subagente con la herramienta Agent, `model: "opus"` |
| 5. Implementar | Sonnet 5.5 | subagente con la herramienta Agent, `model: "sonnet"` |

Los subagentes son `general-purpose`, no heredan esta conversación y solo saben lo que pongas en su prompt: pásales el encargo completo, la ruta exacta del plan y los límites de su fase, y pídeles que te devuelvan su reporte final íntegro. Espera a que terminen antes de seguir. Si la herramienta Agent no está disponible o el modelo pedido no se puede usar, detente en esa fase y repórtalo: no la ejecutes con otro modelo.

Lleva un registro de cada fase (decisión, comandos y resultado): lo necesitas para el cuerpo de la PR y para el reporte final.

## 1. Preparar

1. Separa de `$ARGUMENTS` el encargo y las opciones. Si el encargo es una URL de issue, léela con `gh issue view <url> --json number,title,body,labels,url`; el encargo pasa a ser título + cuerpo y guarda el número para `Closes #<n>`.
2. Ejecuta `git status --short --branch`. Si hay cambios en archivos **versionados** (modificados, borrados o en el índice), detente: no mezcles trabajo ajeno. Los archivos sin seguimiento (por ejemplo otros planes en `plans/`) se toleran, pero nunca se incluyen en el commit.
3. Ejecuta `git fetch <remote> <base>` y, salvo `--upstream none`, `git fetch <upstream> <base>`. Si el remoto propio o su rama no existen, detente. Si falla solo el upstream, continúa y omite la segunda PR.

## 2. Clasificar

Si llega `--tipo`, úsalo. Si no, decide **una** de las tres skills con este orden:

1. Etiquetas de la issue: `bug` → bug; `enhancement` o `feature` → feature; `chore`, `maintenance`, `refactor`, `documentation` o `dependencies` → chore.
2. Contenido del encargo:
   - **bug**: describe un comportamiento actual incorrecto frente a uno esperado (error, fallo, regresión, "no funciona", "debería… pero…").
   - **feature**: pide un comportamiento observable nuevo para el usuario de la aplicación.
   - **chore**: mantenimiento sin cambio de comportamiento observable (refactor, dependencias, CI, documentación, configuración, limpieza).
3. Si sigue siendo ambiguo: bug cuando se describe algo que ya existe y falla; si no, feature.

Anota el tipo elegido y el motivo en una frase.

## 3. Rama

1. Genera un `<slug>` kebab-case en español, sin tildes, de 2 a 5 palabras (p. ej. `limite-titulo-tarea`).
2. Rama según el tipo, con la convención del repositorio: `feat/<slug>`, `fix/<slug>` o `chore/<slug>`. Si la rama ya existe en local o en el remoto, añade el sufijo `-2`, `-3`…
3. Créala desde la base actualizada: `git switch -c <rama> <remote>/<base>`.

## 4. Planificar

1. Ruta del plan: `plans/<tipo>-<slug>.md` (con el mismo sufijo que la rama si lo hubo).
2. Lanza un subagente con `model: "opus"` que invoque la skill `feature`, `bug` o `chore` (herramienta Skill) con argumentos `<encargo> — escribe el plan exactamente en plans/<tipo>-<slug>.md`, que no haga nada más y que devuelva el reporte final de la skill.
3. Ignora la sugerencia final de ejecutar `/implement`: el workflow continúa solo.
4. Comprueba que el plan existe, que empieza por `# Feature:`, `# Bug:` o `# Chore:` según el tipo y que contiene las secciones obligatorias de esa skill. Comprueba también con `git status --short` que la planificación no ha tocado nada más. Si falla cualquiera de las dos cosas, detente.
5. Si el plan deja incertidumbres que impiden implementar con seguridad (secretos, decisiones de producto abiertas, datos que no están en el repositorio), detente aquí y repórtalas.

## 5. Implementar

1. Lanza un subagente con `model: "sonnet"` que invoque la skill `implement` (herramienta Skill) con argumento `plans/<tipo>-<slug>.md`, que no haga commit, push ni PR y que devuelva el reporte final de la skill.
2. Si `implement` informa de un bloqueo o de que el plan está obsoleto, detente: no rediseñes la solución desde este workflow.
3. Guarda la lista de archivos cambiados que reporta; es el alcance del commit.

## 6. Comprobar

De vuelta en este workflow (Opus 5.5), sin delegar: no te fíes del reporte del subagente y ejecuta tú las validaciones de CI (`.github/workflows/ci.yaml`), anotando el resultado de cada una:

```bash
gofmt -l <archivos .go cambiados>   # debe salir vacío
go vet ./...
GO_ENV=test go test -p 1 ./...
go build ./...
git diff --check
```

- Limita `gofmt -l` a los `.go` cambiados: en Windows los finales CRLF hacen que `gofmt -l .` liste archivos ajenos al cambio.
- Si el plan añade migraciones, ejecuta antes `GO_ENV=test soda migrate -e test`.
- Revisa `git status --short` y `git diff`: todo cambio debe estar dentro del alcance del plan. Si aparece algo fuera, detente.
- Si una validación falla por el código, corrige solo dentro del alcance del plan y repite; como máximo dos intentos. Si sigue fallando, detente sin commit y deja la rama tal cual.
- Si los tests no pueden ejecutarse por el entorno (PostgreSQL inaccesible, base `todo_test` inexistente, credenciales), no lo ocultes ni desactives tests: continúa, pero la PR se abrirá como **borrador** y lo indicará.

## 7. Commit

Invoca la skill `git:commit` con argumentos: la lista explícita de archivos cambiados por `implement` (sin el plan, que no se versiona en este repositorio, ni otros archivos sin seguimiento) y un mensaje propuesto con `-m "<tipo>(<scope>): <descripción>"`, Conventional Commits, primera línea ≤ 50 caracteres, tipo `feat`, `fix` o el que corresponda al chore (`refactor`, `docs`, `build`, `ci`, `chore`). Si hay issue, añade `Refs #<n>` en el cuerpo.

Comprueba con `git log -1 --stat` que el commit existe y contiene solo esos archivos. Si no, detente.

## 8. Publicar y abrir las PR

1. `git push -u <remote> <rama>`. Es el único push del workflow.
2. Obtén `OWNER/REPO` de `git remote get-url <remote>` (fork) y de `git remote get-url <upstream>` (original). Pásalos siempre con `--repo`: con varios remotos `gh` podría elegir otro.
3. Escribe el cuerpo en un fichero temporal fuera del repositorio, en español, con:

   - `## Resumen`: qué cambia y por qué, en 2-4 líneas.
   - `## Tipo`: feature, bug o chore y el motivo de la clasificación.
   - `## Cambios`: archivos y responsabilidad.
   - `## Validación`: cada comando con su resultado; lo no ejecutado, marcado y explicado.
   - `## Riesgos`: los del plan que sigan vigentes.
   - La referencia a la issue, si el encargo venía de una: `Closes #<n>` en la PR del repositorio al que pertenece la issue y `Refs <OWNER/REPO>#<n>` en la otra.
   - El plan completo dentro de `<details><summary>Plan</summary>…</details>`.
   - Termina con la línea `🤖 Generated with [Claude Code](https://claude.com/claude-code)`.

4. PR en el fork:

   ```bash
   gh pr create --repo <OWNER/REPO del fork> --base <base> --head <rama> --title "<primera línea del commit>" --body-file <fichero>
   ```

5. PR en el original (salvo `--upstream none` o upstream inaccesible). Antes comprueba que la rama solo aporta los commits de este workflow: `git log --oneline <upstream>/<base>..<rama>` debe mostrar únicamente el commit de la fase 7. Si aparecen más (el `<base>` del fork va por delante del original), **no** abras esta PR y repórtalo, para no arrastrar trabajo ajeno al original. Si está limpio:

   ```bash
   gh pr create --repo <OWNER/REPO original> --base <base> --head <OWNER del fork>:<rama> --title "<primera línea del commit>" --body-file <fichero con la referencia a la issue ajustada y un enlace a la PR del fork>
   ```

6. Usa `--draft` en ambas si alguna validación no pudo ejecutarse.
7. Si ya existe una PR abierta para la rama en alguno de los repositorios (`gh pr list --repo <OWNER/REPO> --head <rama>`), no crees otra: informa de su URL. Si falla la creación de una PR, intenta igualmente la otra y reporta el error.

## Reporte final

Informa de: tipo elegido y motivo, rama, ruta del plan, archivos cambiados, hash y mensaje del commit, resultado de cada validación, URLs de ambas PR (o por qué se omitió la del original, o la fase en la que se detuvo el workflow y por qué) y riesgos pendientes. Al detenerte en cualquier fase, deja el repositorio como esté y di exactamente qué queda hecho y qué no.
