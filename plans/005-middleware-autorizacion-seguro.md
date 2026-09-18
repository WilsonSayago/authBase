# Plan 005: Hacer el middleware de autorización panic-safe

> **Instrucciones para el ejecutor**: este plan depende del TokenManager y los
> puertos de identidad de 003/004. Ejecuta cada gate; no vuelvas a parsear JWT en
> el adaptador HTTP. Actualiza la fila 005 al terminar.
>
> **Drift check inicial**:
> `git diff --stat e43e3ac..HEAD -- core/authorizationusecase.go core/services/authorizationservice.go core/domain`
> Esperar las refactorizaciones 003/004. Si Authorization todavía recibe
> JwtProp o usa MapClaims, los planes previos no están completos: detenerse.

## Estado

- **Prioridad**: P1
- **Esfuerzo**: M
- **Riesgo**: MED
- **Depende de**: planes 003 y 004
- **Categoría**: security, bug
- **Planificado en**: commit `e43e3ac`, 2026-09-17

## Por qué importa

El middleware actual corta el header sin validar longitud ni esquema, hace type
assertions sobre claims/context y devuelve errores internos al cliente. Un
header corto o token firmado sin el claim esperado puede provocar panic. Además
responde 401 ante falta de permisos, aunque la identidad ya fue autenticada.

## Estado actual

`core/services/authorizationservice.go` contiene:

- línea 38: `authHeader[len(BearerSchema):]` sin comprobar prefijo/longitud;
- línea 47: assertion directa a MapClaims y string;
- líneas 50 y 56: errores internos devueltos al cliente;
- línea 68: 401 para autorización insuficiente;
- líneas 75-80: `interface{}(nil).(T)` y `user.(T)`, ambos panicables;
- líneas 61-72: `PoliciesGuard` recibe un callback `func(interface{}, ...)` que
  pierde seguridad de tipos.

Después de 003, Authorization debe recibir un TokenManager. Después de 004, el
puerto de usuario recibe context y las identidades tienen estado activo.

## Comandos

| Propósito | Comando | Resultado esperado |
|---|---|---|
| Tests | `go test ./core/services -run 'TestAuthorizeJWT|TestPoliciesGuard|TestGetUserToken' -count=20` | exit 0 |
| Fuzz | `go test ./core/services -run '^$' -fuzz FuzzAuthorizeHeader -fuzztime=5s` | exit 0, sin panic |
| Race | `go test -race ./...` | exit 0 |
| Búsqueda | `rg -n 'err\.Error\(\)|interface\{\}\(nil\)\.\(T\)|\[len\(BearerSchema\):\]|MapClaims' core/services` | sin coincidencias |

Prefijar comandos Go con
`env GOCACHE=/private/tmp/authbase-go-cache GOWORK=off`.

## Alcance

**Dentro del alcance**:

- `core/authorizationusecase.go`.
- `core/services/authorizationservice.go` y tests.
- Fake Context compartido solo en `_test.go`.
- Cambios pequeños en errores sentinel si 004 ya creó `core/errors.go`.
- `plans/README.md`.

**Fuera del alcance**:

- No introducir una dependencia directa de Gin, Echo, Fiber u otro framework.
- No configurar cookies, CORS o CSRF.
- No cambiar formato JWT ni permisos de dominio.
- No devolver errores de repositorio, parser o criptografía al cliente.
- No llamar `ctx.Next()` después de abortar.

## Flujo Git

- Rama: `codex/005-safe-authorization`.
- Commit sugerido: `security: harden authorization middleware`.

## Pasos

### Paso 1: hacer el contrato Context verificable

Mantener una interfaz framework-neutral, pero añadir solo las operaciones
necesarias para obtener `context.Context` de la request si el puerto de usuario
de 004 lo exige. Documentar una clave de contexto privada/tipada para el usuario
o encapsular Set/Get dentro de helpers; no exportar el string `"user"` como
contrato accidental.

Cambiar `GetUserToken` para devolver `(T, bool)` o `(T, error)`. Nunca fabricar
un nil genérico mediante assertion.

**Verificar**: tests para clave ausente, valor nil, tipo incorrecto y tipo
correcto; ninguno hace panic.

### Paso 2: parsear Bearer de forma estricta

Aceptar exactamente esquema `Bearer` case-insensitive según HTTP, con al menos
un espacio y un token no vacío. Rechazar headers vacíos, cortos, Basic, múltiples
segmentos ambiguos y whitespace-only antes de invocar TokenManager.

Todos los fallos de autenticación deben responder 401 con un error público
estable, sin el token original ni detalles internos. No revelar si falló firma,
expiración, usuario o parser.

**Verificar**: tabla de headers hostiles y fuzz test sin panic.

### Paso 3: validar token e identidad

Usar `TokenManager.ParseAccess`, obtener subject tipado y consultar el puerto de
usuario. Exigir usuario activo conforme a 004. Guardar el usuario en contexto y
llamar `Next()` exactamente una vez solo en éxito.

Contabilizar en el fake: en cualquier error, `Next()` debe quedar en cero y
Abort debe ejecutarse una vez.

**Verificar**: token válido, expirado, refresh usado como access, usuario
ausente/inactivo y error de repositorio.

### Paso 4: corregir PoliciesGuard

- Tipar el callback como `func(T, string, OperationEnum) bool`.
- Si no existe usuario autenticado: 401 y abort.
- Si existe pero no tiene permiso: 403 y abort.
- Admin activo pasa; admin inactivo ya debe haber fallado antes.
- Callback personalizado no debe saltarse el requisito de usuario autenticado y
  activo; solo decide autorización adicional.
- Respuesta pública estable y sin detalles internos.

**Verificar**: tests por matriz identidad × admin × permiso × callback.

## Plan de pruebas

- Header vacío, corto, esquema incorrecto, sin token y con whitespace.
- JWT malformado, firma inválida, tipo incorrecto y expirado.
- Claims nunca causan panic gracias a TokenManager.
- Usuario no encontrado/inactivo/error de repositorio.
- Context sin usuario o con tipo incorrecto.
- 401 para no autenticado; 403 para autenticado sin permiso.
- `Next`/Abort llamados exactamente una vez según el camino.
- Fuzz sobre headers y race detector.

## Criterios de término

- [ ] Ninguna entrada controlada por request puede provocar panic.
- [ ] Authorization no importa `golang-jwt` ni JwtProp.
- [ ] No se envían `err.Error()` ni detalles internos al cliente.
- [ ] 401 y 403 tienen semántica correcta.
- [ ] GetUserToken es seguro ante ausencia o tipo incorrecto.
- [ ] Los callbacks conservan el tipo T y no omiten autenticación.
- [ ] Tests, fuzz breve, race y vet pasan.
- [ ] Fila 005 marcada `DONE`.

## Condiciones de parada

- El Context real de un consumidor no puede suministrar context.Context.
- Un framework obliga a una semántica distinta de Abort/Next y no hay adapter
  disponible para verificarla.
- Se pide exponer razones detalladas de fallo JWT al cliente.
- TokenManager no distingue access de refresh.

## Notas de mantenimiento

Mantener respuestas públicas mínimas y observabilidad interna separada. Si se
añade logging, debe inyectarse y jamás registrar Authorization headers, tokens,
claims completos o claves.
