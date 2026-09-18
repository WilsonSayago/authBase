# Plan 007: Actualizar Go, dependencias y análisis de vulnerabilidades

> **Instrucciones para el ejecutor**: ejecutar después de estabilizar imports.
> No hacer upgrades amplios con `go get -u ./...`; usar versiones exactas. No
> tocar código para ocultar fallos de govulncheck.
>
> **Drift check inicial**:
> `git diff --stat e43e3ac..HEAD -- go.mod go.sum .github/workflows core infra`
> Verificar que initModules ya fue eliminado y los planes 003–006 están verdes.

## Estado

- **Prioridad**: P1
- **Esfuerzo**: S
- **Riesgo**: LOW
- **Depende de**: planes 002–006
- **Categoría**: dependencies, security, dx
- **Planificado en**: commit `e43e3ac`, 2026-09-17
- **Reconciliado con**: cambios no confirmados de `go.mod`/`go.sum`, 2026-09-17

## Por qué importa

El repositorio usa JWT 5.2.1, afectado por GO-2025-3553, x/crypto 0.32.0 y el
toolchain local Go 1.26.3, que govulncheck marcó con cuatro vulnerabilidades
alcanzables. En una copia temporal, JWT 5.3.1, x/crypto 0.57.0 y Go 1.27.1
compilaron, pasaron race/vet y dejaron cero vulnerabilidades alcanzables.

## Estado actual

`go.mod` del árbol de trabajo al planificar:

```go
module github.com/WilsonSayago/authBase
go 1.26.3
require (
    github.com/golang-jwt/jwt/v5 v5.2.1
    golang.org/x/crypto v0.32.0
)
require (
    github.com/WilsonSayago/initModules/v2 v2.0.0 // indirect
    github.com/magiconair/properties v1.18.11 // indirect
    gopkg.in/yaml.v3 v3.0.1 // indirect
)
```

La línea Go 1.26.3 y la migración a initModules v2 son cambios preexistentes no
confirmados frente a HEAD. El plan 001 completa los imports y el plan 002 retira
la dependencia por arquitectura; este plan 007 trabaja sobre el grafo resultante
y no debe deshacer esas decisiones.

Versiones verificadas el 2026-09-17:

- mínimo de lenguaje recomendado: `go 1.26.0` porque x/crypto 0.57.0 lo exige;
- toolchain recomendado: `go1.27.1`;
- `github.com/golang-jwt/jwt/v5 v5.3.1`;
- `golang.org/x/crypto v0.57.0`.

Después del plan 002 no deben existir initModules, properties ni yaml.v3.

## Comandos

| Propósito | Comando | Resultado esperado |
|---|---|---|
| Upgrade | `env GOWORK=off go get github.com/golang-jwt/jwt/v5@v5.3.1 golang.org/x/crypto@v0.57.0` | solo esas dependencias cambian |
| Tidy | `env GOWORK=off go mod tidy` | exit 0 |
| Tests | `env GOCACHE=/private/tmp/authbase-go-cache GOWORK=off GOTOOLCHAIN=go1.27.1 go test -race ./...` | exit 0 |
| Vet | `env GOCACHE=/private/tmp/authbase-go-cache GOWORK=off GOTOOLCHAIN=go1.27.1 go vet ./...` | exit 0 |
| Vulnerabilidades | `env GOCACHE=/private/tmp/authbase-go-cache GOWORK=off GOTOOLCHAIN=go1.27.1 go run golang.org/x/vuln/cmd/govulncheck@v1.8.0 ./...` | “No vulnerabilities found” y exit 0 |

## Alcance

**Dentro del alcance**:

- `go.mod`, `go.sum`.
- `.github/workflows/ci.yml` del plan 001.
- Documentar política de Go en README solo si 008 todavía no comenzó; si no,
  dejarlo para 008.
- `plans/README.md`.

**Fuera del alcance**:

- No actualizar dependencias inexistentes o herramientas globales.
- No reintroducir initModules después de que el plan 002 haya retirado el ciclo
  de vida singleton de authBase.
- No cambiar module path.
- No añadir `replace`.
- No ignorar avisos alcanzables mediante excludes.

## Flujo Git

- Rama: `codex/007-upgrade-dependencies`.
- Commit sugerido: `build: update secure Go dependency baseline`.

## Pasos

### Paso 1: confirmar grafo previo

Ejecutar `go mod graph` y `go list -m all`. Confirmar que initModules,
properties y yaml.v3 ya no están. Si aparecen, detenerse y corregir el plan 002,
no ocultarlos con excludes.

### Paso 2: fijar política Go

Usar:

```go
go 1.26.0
toolchain go1.27.1
```

La directiva `go` expresa compatibilidad mínima; el toolchain expresa la versión
de desarrollo recomendada. CI debe probar el último parche de 1.26 y 1.27. No
fijar `go 1.27.1` salvo que el código use una característica exclusiva de 1.27.

**Verificar**: `GOTOOLCHAIN=go1.26.8 go test ./...` y
`GOTOOLCHAIN=go1.27.1 go test ./...` → exit 0.

### Paso 3: actualizar versiones exactas

Ejecutar el `go get` exacto y `go mod tidy`. Revisar el diff: solamente JWT,
x/crypto y sus transitivas legítimas pueden cambiar.

**Verificar**: `go list -m all` contiene las versiones objetivo.

### Paso 4: añadir gate de vulnerabilidades

En CI, ejecutar govulncheck v1.8.0 con Go 1.27.1. Mantener test/race/vet del plan
001. Fijar cualquier GitHub Action por SHA.

**Verificar**: ejecución local devuelve cero vulnerabilidades alcanzables.

### Paso 5: verificar reproducibilidad

Ejecutar:

```sh
env GOWORK=off go mod verify
env GOWORK=off GOTOOLCHAIN=go1.26.8 go test ./...
env GOWORK=off GOTOOLCHAIN=go1.27.1 go test -race ./...
env GOWORK=off GOTOOLCHAIN=go1.27.1 go vet ./...
```

Todos deben terminar en 0.

## Plan de pruebas

- Suite completa en Go 1.26.8 y 1.27.1.
- Race/vet/govulncheck en 1.27.1.
- `go mod verify`.
- Confirmar que no se modificaron APIs ni archivos productivos.

## Criterios de término

- [ ] JWT 5.3.1 y x/crypto 0.57.0 fijados.
- [ ] `go 1.26.0` y `toolchain go1.27.1` documentados.
- [ ] No quedan dependencias de initModules.
- [ ] Tests pasan en Go 1.26.8 y 1.27.1.
- [ ] govulncheck informa cero vulnerabilidades alcanzables.
- [ ] CI ejecuta govulncheck y acciones están pinneadas.
- [ ] Fila 007 marcada `DONE`.

## Condiciones de parada

- x/crypto o JWT requieren cambios productivos no previstos.
- Algún consumidor exige Go 1.25 o anterior.
- govulncheck encuentra un advisory alcanzable después del upgrade.
- `go mod tidy` añade nuevamente dependencias retiradas.

## Notas de mantenimiento

Revisar parches Go mensualmente y dependencias semanalmente. La directiva Go no
reemplaza actualizar el toolchain que compila cada aplicación consumidora.
