# Plan 008: Preparar la API y publicación válida de authBase v3

> **Instrucciones para el ejecutor**: este plan prepara una release pero no la
> publica. No crear, mover o eliminar tags; no hacer push. Requiere todos los
> planes anteriores en DONE y una decisión explícita del mantenedor sobre
> licencia y compatibilidad.
>
> **Drift check inicial**:
> `git diff --stat e43e3ac..HEAD -- . ':!plans'`
> Revisar los cambios de 001–007. Si alguno sigue TODO/BLOCKED, detenerse.

## Estado

- **Prioridad**: P2
- **Esfuerzo**: L
- **Riesgo**: HIGH
- **Depende de**: planes 001–007
- **Categoría**: migration, docs, dx
- **Planificado en**: commit `e43e3ac`, 2026-09-17

## Por qué importa

Las etiquetas v2.0.0–v2.0.3 existentes no son instalables porque el module path
no termina en `/v2`. Las etiquetas v1 pertenecen históricamente al módulo
`github.com/WilsonSayago/middleware`, y `@main` solo resuelve mediante una
pseudo-versión. Los cambios de esta serie rompen APIs; deben publicarse con una
identidad nueva y válida, recomendada como v3.

## Estado actual

- `go.mod:1`: `module github.com/WilsonSayago/authBase`.
- Todos los imports internos usan `github.com/WilsonSayago/authBase/...`.
- `git remote -v` durante la auditoría apuntó a
  `https://github.com/WilsonSayago/middleware.git`, posiblemente mediante un
  redirect; esto debe resolverse antes de preparar release.
- `README.md` contiene solo título y nombre.
- No hay LICENSE, guía de migración, changelog ni documentación de API.
- Go exige sufijo de major version desde v2. Como v2 ya fue etiquetado de forma
  inválida, no mover esas etiquetas; preparar `v3.0.0` con `/v3`.

## Comandos

| Propósito | Comando | Resultado esperado |
|---|---|---|
| Imports antiguos | `rg -n 'github\.com/WilsonSayago/authBase(?!/v3)' --pcre2 --glob '*.go' go.mod` | sin coincidencias al terminar |
| Tests | `env GOCACHE=/private/tmp/authbase-go-cache GOWORK=off go test -race ./...` | exit 0 |
| Vet | `env GOCACHE=/private/tmp/authbase-go-cache GOWORK=off go vet ./...` | exit 0 |
| Vulnerabilidades | `env GOWORK=off go run golang.org/x/vuln/cmd/govulncheck@v1.8.0 ./...` | 0 alcanzables |
| Listado | `env GOWORK=off go list ./...` | paquetes bajo `github.com/WilsonSayago/authBase/v3/...` |
| Diff | `git diff --check` | sin salida |

## Alcance

**Dentro del alcance**:

- `go.mod`, todos los imports internos Go y `go.sum` si cambia.
- Revisar y estabilizar interfaces públicas modificadas en 003–006.
- `README.md`.
- Crear `CHANGELOG.md`.
- Crear `docs/MIGRATION_V3.md`, `docs/SECURITY.md` y ejemplos compilables.
- `.github/workflows/ci.yml` para añadir test de ejemplos y API.
- Añadir LICENSE solo después de que el mantenedor elija explícitamente una.
- `plans/README.md`.

**Fuera del alcance**:

- No crear tags, releases, PRs ni hacer push.
- No borrar ni mover tags v1/v2 existentes.
- No cambiar la URL remote automáticamente.
- No prometer compatibilidad con tokens pre-v3 si no existe transición probada.
- No implementar nuevos features durante el freeze de API.

## Flujo Git

- Rama: `codex/008-authbase-v3`.
- Commits: `refactor: adopt authBase v3 module path` y
  `docs: prepare v3 migration and release`.

## Pasos

### Paso 1: resolver identidad canónica y licencia

Confirmar con el mantenedor:

1. repositorio canónico: `github.com/WilsonSayago/authBase`;
2. major objetivo: v3;
3. licencia elegida;
4. política de compatibilidad de tokens y API.

Si el remote continúa apuntando a `middleware.git`, no cambiarlo: informar para
que el operador lo corrija explícitamente. Si no hay decisión de licencia, se
puede continuar documentación técnica pero no declarar release-ready.

### Paso 2: congelar API pública

Ejecutar `go doc` por paquete y revisar cada identificador exportado. La API v3
debe cumplir al menos:

- constructores retornan errores y nunca singletons;
- TokenManager es el único dueño de JWT;
- interfaces I/O aceptan context.Context;
- IUserGeneric no expone credenciales;
- estado activo y errores sentinel están documentados;
- autorización diferencia 401/403;
- refresh exige RefreshTokenStore atómico;
- no existe `ValidateTokenAndRefresh` incompleto;
- nombres exportados siguen estilo Go (`ID`, `JWT`), sin `interface{}` cuando
  un genérico concreto es posible.

No hacer cambios nuevos de diseño: si un punto no se cumple, regresar el plan
responsable a BLOCKED y detener este.

### Paso 3: adoptar module path v3

Cambiar a:

```go
module github.com/WilsonSayago/authBase/v3
```

Actualizar todos los imports internos al prefijo `/v3`. No usar una carpeta
física `v3/` salvo que el mantenedor decida mantener v1 en la misma rama; la
opción preferida es major branch/root module.

**Verificar**: búsqueda de imports antiguos sin resultados, `go list ./...` y
suite completa exitosos.

### Paso 4: escribir README utilizable

README debe incluir:

- propósito y límites de authBase;
- versión mínima/recomendada de Go;
- instalación futura `.../authBase/v3` sin afirmar que la etiqueta ya existe;
- ejemplo completo de config validada, puertos, construcción, login,
  middleware y refresh store;
- modelo de amenazas: Bearer tokens, almacenamiento de secretos, replay,
  revocación, no logging;
- contrato de errores y 401/403;
- comandos test/race/vet/govulncheck;
- estado de compatibilidad y enlace a migración.

Los ejemplos deben vivir en `examples/` como módulos o paquetes compilables, no
solo bloques Markdown no verificados.

### Paso 5: documentar migración y seguridad

`docs/MIGRATION_V3.md` debe mapear API antigua → nueva, incluyendo:

- eliminación de singletons/initModules;
- nuevos constructores con error;
- claims/tokens incompatibles;
- separación de credenciales;
- context.Context;
- RefreshTokenStore y atomicidad;
- cambios de middleware y errores.

`docs/SECURITY.md` debe indicar cómo reportar vulnerabilidades sin publicarlas,
versiones soportadas y qué datos nunca loggear. No inventar un email: usar un
canal confirmado por el mantenedor o dejar un marcador que bloquee la release.

### Paso 6: changelog y checklist de release

Crear CHANGELOG con sección `Unreleased` y entrada v3 preparada, sin fecha ni
afirmar que fue publicada. Añadir checklist documental que exija:

- working tree limpio;
- CI verde en dos Go versions;
- tests/race/vet/govulncheck;
- ejemplos compilables;
- `go list -m` del path `/v3` usando un tag de prueba local o validación
  equivalente;
- licencia y security contact confirmados;
- revisión manual de migración.

### Paso 7: probar como consumidor externo

En un directorio temporal fuera del repo, crear un módulo consumidor con
`replace github.com/WilsonSayago/authBase/v3 => <ruta-local>` y compilar el
quickstart. Esto valida imports públicos sin publicar.

**Verificar**: `go test ./...` en el consumidor temporal → exit 0. No guardar el
replace en authBase.

## Plan de pruebas

- Suite/race/vet/govulncheck completa.
- Compilación de todos los ejemplos.
- Módulo consumidor temporal sin acceso a internals.
- Búsqueda de imports antiguos.
- Revisión de `go doc` y breaking API documentada.
- No existe ningún comando de publicación dentro de CI o scripts.

## Criterios de término

- [ ] Module path e imports usan `/v3`.
- [ ] API pública cumple la lista de freeze.
- [ ] README, migración, seguridad y changelog están completos.
- [ ] Ejemplos y consumidor temporal compilan.
- [ ] CI, race, vet y govulncheck pasan.
- [ ] Licencia y security contact están confirmados o el plan queda BLOCKED.
- [ ] No se creó ningún tag/release/push.
- [ ] Fila 008 marcada `DONE` o `BLOCKED` con motivo exacto.

## Condiciones de parada

- El repositorio canónico no es `github.com/WilsonSayago/authBase`.
- El mantenedor prefiere recuperar una línea v1 en vez de v3.
- Falta elección explícita de licencia o canal de seguridad al declarar
  release-ready.
- Algún plan 001–007 no está DONE.
- Los consumidores requieren continuidad de tokens/API no cubierta.

## Notas de mantenimiento

Una vez publicado un tag, no moverlo ni reutilizarlo. La publicación real debe
ser una acción separada y explícitamente autorizada. Mantener v3 compatible; el
siguiente breaking change exige `/v4`.
