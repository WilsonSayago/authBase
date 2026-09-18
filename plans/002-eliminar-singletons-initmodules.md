# Plan 002: Eliminar singletons globales e initModules

> **Instrucciones para el ejecutor**: sigue cada paso y ejecuta sus gates. Ante
> una condición de parada, informa y no improvises. Actualiza la fila 002 del
> índice al terminar.
>
> **Drift check inicial**:
> `git diff --stat e43e3ac..HEAD -- core/services infra/secundary go.mod go.sum`
> Reconciliar únicamente los cambios introducidos por el plan 001. Cualquier
> otro cambio en constructores o dependencias es una condición de parada.

## Estado

- **Prioridad**: P1
- **Esfuerzo**: M
- **Riesgo**: MED
- **Depende de**: `plans/001-establecer-pruebas-y-ci.md`
- **Categoría**: bug, tech-debt, migration
- **Planificado en**: commit `e43e3ac`, 2026-09-17
- **Reconciliado con**: `initModules/v2 v2.0.0`, 2026-09-17

## Por qué importa

`initModules/v2 v2.0.0` corrige el defecto de v1 que compartía un único
`sync.Once`: ahora `GetInstance` es concurrente y mantiene un `sync.Once` por
clave. Sin embargo, la propia v2 marca esa API como deprecated porque las claves
string son globales al proceso. En authBase, una misma clave conserva para
siempre el primer puerto/configuración y colisiona entre instanciaciones
genéricas. Una librería reutilizable no debe decidir el ciclo de vida de las
dependencias de la aplicación consumidora.

## Estado actual

Patrón repetido en `core/services/authenticationservice.go:20-28`:

```go
instance := initModules.GetInstance("AuthenticationService", func() interface{} {
    return &AuthenticationService[T]{port: port, validatePort: validatePort, prop: prop}
})
return instance.(*AuthenticationService[T])
```

El mismo patrón aparece en:

- `core/services/authorizationservice.go:19-26`;
- `core/services/role_service.go:14-20`;
- `infra/secundary/validationservice.go:11-15`.

Después del plan 001, los cuatro archivos importan
`github.com/WilsonSayago/initModules/v2` y `go.mod` la requiere directamente en
v2.0.0. `magiconair/properties` y `gopkg.in/yaml.v3` son transitivas de esa
dependencia y no se importan directamente desde authBase.

La alternativa v2 `Once[T]` sigue siendo global por tipo, y `OnceIn` con un
`Container` solo sería apropiada si el contenedor perteneciera al composition
root del consumidor. Este plan no sustituye un singleton por otro: devuelve
instancias normales y deja que cada aplicación decida si las comparte.

## Comandos

| Propósito | Comando | Resultado esperado |
|---|---|---|
| Buscar uso | `rg -n 'initModules|GetInstance\(' --glob '*.go'` | cero referencias productivas al terminar |
| Tidy | `env GOWORK=off go mod tidy` | exit 0; elimina dependencias no usadas |
| Tests | `env GOCACHE=/private/tmp/authbase-go-cache GOWORK=off go test -race ./...` | exit 0 |
| Vet | `env GOCACHE=/private/tmp/authbase-go-cache GOWORK=off go vet ./...` | exit 0 |
| Diff | `git diff --check` | sin salida |

## Alcance

**Dentro del alcance**:

- `core/services/authenticationservice.go`.
- `core/services/authorizationservice.go`.
- `core/services/role_service.go`.
- `infra/secundary/validationservice.go`.
- Tests de constructores en esos dos paquetes.
- `go.mod` y `go.sum` mediante `go mod tidy`.
- `plans/README.md`.

**Fuera del alcance**:

- No cambiar firmas públicas todavía.
- No validar configuración JWT; corresponde al plan 003.
- No actualizar JWT, x/crypto ni la directiva Go; corresponde al plan 007.
- No migrar a `Once`, `OnceIn`, `OnceValue` ni `Container` dentro de authBase.
- No crear otro registry, singleton o variable global equivalente.

## Flujo Git

- Rama: `codex/002-remove-singletons`.
- Commit sugerido: `refactor: remove global service singletons`.
- Preservar el cambio preexistente de la directiva Go; `go mod tidy` no debe
  retrocederla ni incrementarla.

## Pasos

### Paso 1: añadir tests de identidad de instancias

Crear tests que fallen con v2.0.0 por el comportamiento global y comprueben que:

- dos llamadas al mismo constructor devuelven punteros distintos;
- cada instancia conserva sus propios puertos y configuración;
- dos instanciaciones del genérico con tipos diferentes no colisionan ni hacen
  panic por reutilizar la clave `AuthenticationService`;
- construir concurrentemente instancias con dependencias distintas no conserva
  silenciosamente la dependencia de la primera llamada.

Usar fakes del plan 001 y pruebas dentro del mismo paquete cuando sea necesario
inspeccionar campos privados.

**Verificar antes del cambio**: los tests de regresión deben fallar por la causa
esperada, no por error de compilación. Después continuar inmediatamente; no
committear una suite roja.

### Paso 2: convertir factories en constructores normales

Mantener las firmas actuales pero retornar una nueva instancia directamente:

- `GetAuthenticationInstance` → `&AuthenticationService[T]{...}`;
- `NewAuthorization` → `&Authorization[T,C]{...}`;
- `GetRoleServiceInstance` → `&RoleService{...}`;
- `NewValidationService` → `&ValidationService{}`.

Eliminar imports de initModules. No introducir estado de paquete.

**Verificar**:
`go test ./core/services ./infra/secundary -run 'Test.*Instance|Test.*Constructor' -count=20`
→ exit 0.

### Paso 3: retirar la dependencia

Eliminar el require directo de initModules y ejecutar `go mod tidy`. Confirmar
que `properties` y `yaml.v3` desaparecen si ninguna referencia real permanece.

**Verificar**:

```sh
rg -n 'initModules|magiconair|yaml\.v3' go.mod go.sum core infra
```

→ sin coincidencias productivas.

### Paso 4: ejecutar la suite completa

Ejecutar tests, race, vet y diff check. Revisar `git status --short` y separar
los `.DS_Store` y cambios previos del commit.

## Plan de pruebas

- Instancias repetidas con dependencias distintas.
- Tipos genéricos distintos.
- Ejecución concurrente de 100 construcciones: todas deben ser independientes y
  el race detector debe quedar limpio.
- Mantener todos los tests del plan 001.

## Criterios de término

- [ ] No hay imports ni requires de initModules.
- [ ] No existen registries o singletons nuevos.
- [ ] Cada constructor devuelve una instancia independiente.
- [ ] `go mod tidy` dejó un grafo mínimo.
- [ ] `go test -race ./...` y `go vet ./...` pasan.
- [ ] No se alteró la API pública ni la directiva Go.
- [ ] Fila 002 marcada `DONE`.

## Condiciones de parada

- Algún consumidor vive dentro del repositorio y depende de identidad singleton.
- `initModules` aparece en archivos distintos de los enumerados.
- `go mod tidy` intenta cambiar la directiva Go o añadir dependencias inesperadas.
- Los tests requieren acceso a red o repositorios reales.

## Notas de mantenimiento

El composition root de cada aplicación consumidora debe decidir si comparte
instancias. Un reviewer debe rechazar cualquier intento de reintroducir un
singleton para “preservar compatibilidad”.
