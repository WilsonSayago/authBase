# Plan 001: Completar initModules/v2 y establecer pruebas y CI

> **Instrucciones para el ejecutor**: sigue este plan paso a paso. Ejecuta cada
> verificación y confirma el resultado antes de continuar. Si ocurre una
> condición de parada, detente e informa; no improvises. Al terminar actualiza
> la fila 001 de `plans/README.md`, salvo que el revisor mantenga el índice.
>
> **Drift check inicial**:
> `git diff --stat e43e3ac..HEAD -- core infra .github go.mod go.sum`
> Si cualquier archivo productivo cambió, compara el estado actual descrito
> aquí y detente si las firmas ya no coinciden.

## Estado

- **Prioridad**: P1
- **Esfuerzo**: M
- **Riesgo**: LOW
- **Depende de**: ninguno
- **Categoría**: tests, dx
- **Planificado en**: commit `e43e3ac`, 2026-09-17
- **Reconciliado con**: cambios no confirmados de `go.mod`/`go.sum`, 2026-09-17

## Por qué importa

El repositorio tiene 582 líneas Go y cero archivos `_test.go`. El usuario ya
añadió `initModules/v2 v2.0.0` al manifest, pero los imports productivos todavía
apuntan a v1 y la compilación aislada falla. Este plan primero completa esa
migración mínima y luego crea una línea base verde sin codificar como correcto
ninguno de los comportamientos inseguros que los planes posteriores eliminarán.

## Estado actual

- `core/domain/userGeneric.go:50-93` agrega permisos de todos los roles y niega
  operaciones desconocidas.
- `infra/secundary/validationservice.go:18-28` envuelve bcrypt.
- `core/services/authenticationservice.go:77-95` implementa el happy path de
  login.
- `go.mod` requiere `github.com/WilsonSayago/initModules/v2 v2.0.0` como
  indirecta, mientras `authenticationservice.go`, `authorizationservice.go`,
  `role_service.go` y `validationservice.go` todavía importan
  `github.com/WilsonSayago/initModules`. Con `GOWORK=off`, `go test ./...`
  falla con `no required module provides package`.
- El tag v2.0.0 descargado declara correctamente
  `module github.com/WilsonSayago/initModules/v2`; no necesita `replace`.
- No existen `.github/workflows`, `Makefile` ni tests.
- No hay convención de tests previa. Usar solo `testing` estándar, tests en
  tabla y fakes pequeños dentro de archivos `_test.go`.
- No existe una convención estable de commits; usar mensajes claros como
  `test: establish authentication baseline`.

## Comandos

| Propósito | Comando | Resultado esperado |
|---|---|---|
| Migración | `rg -n 'github.com/WilsonSayago/initModules"' --glob '*.go'` | sin imports v1 al terminar |
| Grafo | `env GOWORK=off go list -m all` | contiene `initModules/v2 v2.0.0`, no contiene initModules v1 |
| Tests | `env GOCACHE=/private/tmp/authbase-go-cache GOWORK=off go test ./...` | exit 0; ya no todos los paquetes muestran `[no test files]` |
| Race | `env GOCACHE=/private/tmp/authbase-go-cache GOWORK=off go test -race ./...` | exit 0 |
| Vet | `env GOCACHE=/private/tmp/authbase-go-cache GOWORK=off go vet ./...` | exit 0 |
| Formato | `gofmt -l core infra` | sin salida |
| Diff | `git diff --check` | sin salida |

## Alcance

**Dentro del alcance**:

- Cambiar únicamente el import de initModules a `/v2` en
  `core/services/authenticationservice.go`,
  `core/services/authorizationservice.go`, `core/services/role_service.go` e
  `infra/secundary/validationservice.go`.
- `go.mod` y `go.sum`, exclusivamente mediante `go mod tidy` después de esos
  imports.
- Crear `core/domain/userGeneric_test.go`.
- Crear `infra/secundary/validationservice_test.go`.
- Crear `core/services/test_helpers_test.go`.
- Crear `core/services/authenticationservice_test.go`.
- Crear `.github/workflows/ci.yml`.
- Actualizar `plans/README.md`.

**Fuera del alcance**:

- No modificar código productivo fuera de los cuatro imports indicados.
- No cambiar versiones, el module path de authBase ni el comportamiento de los
  constructores singleton.
- No migrar de `GetInstance` a `Once`, `OnceIn` o `Container`; el plan 002
  elimina ese ciclo de vida de la librería.
- No probar que un `panic` actual sea comportamiento deseado.
- No añadir testify, gomock u otra dependencia de tests.
- No añadir todavía `govulncheck` como gate: las versiones actuales fallan y
  serán actualizadas por el plan 007.

## Flujo Git

- Rama sugerida: `codex/001-test-baseline`.
- Dos commits lógicos si el operador desea conservar trazabilidad:
  `build: complete initModules v2 migration` y
  `test: establish authentication baseline`.
- No incluir `.DS_Store`. Los cambios de `go.mod`/`go.sum` de initModules v2 sí
  pertenecen a esta migración; preservar la directiva Go `1.26.3` sin alterarla.

## Pasos

### Paso 1: completar la migración de imports a v2

En los cuatro archivos productivos enumerados en Alcance, cambiar solamente:

```go
"github.com/WilsonSayago/initModules"
```

por:

```go
"github.com/WilsonSayago/initModules/v2"
```

Mantener el identificador de paquete `initModules`. Ejecutar `go mod tidy` con
`GOWORK=off`; la dependencia v2 debe pasar a directa, y las sumas huérfanas de
v1.0.6 deben desaparecer. No añadir `replace`.

**Verificar**:

```sh
env GOWORK=off go mod tidy
env GOCACHE=/private/tmp/authbase-go-cache GOWORK=off go test ./...
env GOWORK=off go list -m all
```

→ todos exit 0; el listado contiene `initModules/v2 v2.0.0` y no contiene el
módulo v1.

### Paso 2: probar el dominio de permisos

En `core/domain/userGeneric_test.go`, escribir tests en tabla para:

1. constructor y getters públicos;
2. unión OR de permisos duplicados para la misma entidad;
3. permisos de entidades distintas;
4. ausencia de permiso y operación desconocida;
5. bandera de administrador y estado `Active` almacenados correctamente.

No añadir todavía expectativas sobre roles inactivos; ese contrato cambia en
el plan 004.

**Verificar**:
`env GOCACHE=/private/tmp/authbase-go-cache GOWORK=off go test ./core/domain -run Test -count=1`
→ exit 0.

### Paso 3: probar bcrypt

En `infra/secundary/validationservice_test.go`, probar:

- hash distinto del texto de entrada;
- round-trip válido;
- contraseña incorrecta;
- contraseña bcrypt inválida;
- longitud superior al límite de bcrypt devuelve error desde `HashPassword`.

No imprimir contraseñas ni hashes en mensajes de test.

**Verificar**:
`env GOCACHE=/private/tmp/authbase-go-cache GOWORK=off go test ./infra/secundary -run TestValidation -count=1`
→ exit 0.

### Paso 4: crear fakes reutilizables y happy paths

En `core/services/test_helpers_test.go`, definir implementaciones mínimas de
`GenericPort`, `ValidationPort` y, si hace falta, del usuario genérico. Mantener
los fakes privados al paquete y permitir contabilizar llamadas.

En `authenticationservice_test.go`, construir `AuthenticationService` de forma
directa dentro del paquete `services`, evitando los constructores singleton.
Cubrir:

- login válido devuelve dos tokens no vacíos;
- contraseña incorrecta no genera tokens;
- error de `FindByEmail` se propaga como error;
- `ValidateToken` recupera el usuario correcto para un access token emitido por
  el servicio;
- token firmado con otra clave es rechazado.

No añadir casos de refresh, algoritmos alternativos, claims ausentes ni estado
activo: esos tests pertenecen a 003 y 004.

**Verificar**:
`env GOCACHE=/private/tmp/authbase-go-cache GOWORK=off go test ./core/services -run 'Test(Authentication|Login|ValidateToken)' -count=1`
→ exit 0.

### Paso 5: añadir CI mínima

Crear `.github/workflows/ci.yml` para pull requests y pushes a `main` con:

- checkout por SHA estable;
- matriz Go `1.26.8` y `1.27.1`;
- `go test ./...`;
- `go test -race ./...` únicamente en `1.27.1` para no duplicar costo;
- `go vet ./...`;
- cache de módulos provista por `actions/setup-go`.

No usar `@main` en acciones. Si no se dispone de los SHA correctos de acciones,
detenerse y pedir autorización antes de fijarlos.

**Verificar**: validar sintaxis YAML y ejecutar localmente los tres comandos de
la sección Comandos → todos exit 0.

## Plan de pruebas

- Tests nuevos: dominio, bcrypt y happy path de autenticación.
- Todos deben ser deterministas, sin red, filesystem ni reloj real salvo la
  emisión JWT existente.
- Ejecutar cada paquete con `-count=20` para detectar dependencia de orden:
  `go test ./core/domain ./core/services ./infra/secundary -count=20` → exit 0.

## Criterios de término

- [ ] Existen al menos los cuatro archivos `_test.go` indicados.
- [ ] Los cuatro imports usan `/v2`; `go list -m all` no contiene initModules
      v1 y no existe ningún `replace`.
- [ ] Los cinco grupos de casos descritos pasan.
- [ ] `go test ./...`, `go test -race ./...` y `go vet ./...` terminan en 0.
- [ ] CI usa dos toolchains y acciones fijadas por SHA.
- [ ] Los únicos cambios productivos previos a los tests son los cuatro imports
      y el tidy esperado de `go.mod`/`go.sum`.
- [ ] `git diff --check` no informa errores.
- [ ] La fila 001 de `plans/README.md` está en `DONE`.

## Condiciones de parada

- Los constructores o interfaces actuales no coinciden con los extractos.
- `initModules/v2@v2.0.0` deja de resolver o su `go.mod` no declara `/v2`.
- Un test solo puede pasar afirmando un `panic` o una vulnerabilidad como
  comportamiento correcto.
- CI requiere secretos, servicios externos o permisos adicionales.
- Se necesita modificar producción para hacer posible el baseline.

## Notas de mantenimiento

Los tests de este plan describen solo happy paths estables. Los planes 002–006
deben ampliar la suite antes de cambiar comportamiento. Revisar que ningún
agente elimine casos existentes para acomodar una refactorización.
