# Plan 006: Implementar rotación y detección de replay de refresh tokens

> **Instrucciones para el ejecutor**: este es un cambio de arquitectura de alto
> riesgo. Implementa el puerto y la máquina de estados exactamente como se
> describe; no añadas un almacenamiento concreto. Si el consumidor no puede
> garantizar rotación atómica, detente.
>
> **Drift check inicial**:
> `git diff --stat e43e3ac..HEAD -- core/authenticationusecase.go core/services core/port core/domain`
> Los planes 003 y 004 deben estar DONE. TokenManager debe emitir claims tipados
> y los puertos deben usar context.Context.

## Estado

- **Prioridad**: P2
- **Esfuerzo**: L
- **Riesgo**: HIGH
- **Depende de**: planes 003 y 004
- **Categoría**: security, architecture
- **Planificado en**: commit `e43e3ac`, 2026-09-17

## Por qué importa

El refresh actual es completamente stateless: cualquier token robado puede
reutilizarse hasta expirar y los tokens de un usuario comparten identificador.
Rotar solo el JWT sin consumir el anterior no evita replay. authBase debe definir
un puerto con semántica transaccional para que cada aplicación conecte su DB.

## Estado actual esperado después de 003/004

- TokenManager distingue access y refresh, emite `jti` aleatorio y valida
  issuer/audience/tipo/exp.
- AuthenticationService revalida al usuario antes de refrescar.
- La API todavía devuelve un par nuevo sin registrar ni consumir el refresh
  anterior.

No usar el `ID` de usuario como `jti`. La familia de refresh es un identificador
independiente compartido por las rotaciones de una sesión.

## Contrato objetivo

Crear tipos de dominio equivalentes a:

```go
type RefreshSession struct {
    TokenID   string
    FamilyID  string
    UserID    string
    TokenHash [32]byte
    IssuedAt  time.Time
    ExpiresAt time.Time
}
```

Y un puerto cuya operación de rotación sea atómica:

```go
type RefreshTokenStore interface {
    Create(ctx context.Context, session RefreshSession) error
    Rotate(ctx context.Context, currentHash [32]byte, next RefreshSession) error
    RevokeFamily(ctx context.Context, familyID string) error
}
```

Definir sentinels como `ErrRefreshNotFound`, `ErrRefreshConsumed` y
`ErrRefreshFamilyRevoked`. `Rotate` debe consumir el actual y crear el siguiente
en una sola transacción. Ajustar nombres si el código vigente ya estableció una
convención, pero preservar estas garantías.

## Comandos

| Propósito | Comando | Resultado esperado |
|---|---|---|
| Refresh tests | `go test ./core/services -run 'TestRefresh|TestTokenFamily' -count=20` | exit 0 |
| Race | `go test -race ./core/services -run 'TestConcurrentRefresh' -count=20` | exit 0 |
| Suite | `go test -race ./...` | exit 0 |
| Vet | `go vet ./...` | exit 0 |

Prefijar con `env GOCACHE=/private/tmp/authbase-go-cache GOWORK=off`.

## Alcance

**Dentro del alcance**:

- Crear `core/domain/refresh_session.go` y tests.
- Crear `core/port/refresh_token_port.go`.
- `core/services/token_manager.go`, `authenticationservice.go` y tests.
- `core/authenticationusecase.go`.
- Fakes transaccionales únicamente en tests.
- Documento corto `docs/refresh-token-store.md` con garantías para adapters.
- `plans/README.md`.

**Fuera del alcance**:

- No implementar PostgreSQL, MongoDB, Redis ni almacenamiento en memoria para
  producción.
- No almacenar el JWT refresh en texto plano.
- No usar listas globales o maps de proceso como revocación.
- No introducir fallback no atómico.
- No definir logout HTTP; exponer una operación de revocación de familia para
  que cada adapter la use.

## Flujo Git

- Rama: `codex/006-refresh-rotation`.
- Commits: `feat: define refresh token store` y
  `security: rotate refresh token families`.

## Pasos

### Paso 1: definir dominio, errores y puerto

Añadir RefreshSession sin datos de perfil ni secretos. El hash debe ser SHA-256
del token serializado completo y calcularse antes de llamar al store. Documentar
que el store guarda hash, nunca token plano.

El contrato de Rotate debe especificar:

- éxito solo si el actual existe, pertenece a familia activa, no fue consumido
  y no expiró;
- consumo y creación siguiente en una transacción;
- dos llamadas concurrentes: exactamente una gana;
- detectar token consumido permite revocar toda la familia.

**Verificar**: tests del fake reproducen estas invariantes.

### Paso 2: ampliar claims de refresh

Añadir family ID únicamente a refresh claims. Access tokens no deben contenerlo
salvo una justificación documentada. TokenManager debe permitir emitir el primer
refresh con familia nueva y el siguiente conservando familia pero con jti nuevo.

**Verificar**: access carece de familia; dos refresh consecutivos tienen la misma
familia y diferentes jti/hash.

### Paso 3: registrar login y retirar GetToken público

La emisión inicial tras login debe:

1. validar credenciales y estado;
2. emitir par con family/jti aleatorios;
3. crear RefreshSession mediante el store;
4. devolver tokens solo si el store confirmó persistencia.

Convertir el helper genérico `GetToken(id)` en operación privada `issuePair` o
equivalente; no permitir emitir refresh sin registrarlo. Actualizar interfaz.

**Verificar**: fallo de store no devuelve tokens y conserva el error mediante
wrapping.

### Paso 4: rotar de forma atómica

Refresh debe:

1. parsear/validar token tipado;
2. revalidar usuario activo;
3. calcular hash del token recibido;
4. preparar siguiente sesión/token;
5. llamar Rotate una vez;
6. devolver el par nuevo solo después de éxito.

Ante `ErrRefreshConsumed`, intentar `RevokeFamily` y devolver un error público de
refresh inválido. Si revocar falla, preservar ambas causas para observabilidad
sin exponerlas al cliente.

**Verificar**: reutilizar token anterior revoca la familia; token siguiente ya
no puede rotar.

### Paso 5: probar concurrencia y cancelación

Dos goroutines refrescando el mismo token deben producir exactamente un éxito y
un replay. Context cancelado debe detener operaciones del store y nunca devolver
un token no persistido.

**Verificar**: race detector y test concurrente 20 veces.

### Paso 6: documentar adapter contract

`docs/refresh-token-store.md` debe especificar esquema lógico, índices únicos
sugeridos sobre hash/jti, atomicidad y retención/limpieza. No incluir SQL ni una
tecnología concreta.

## Plan de pruebas

- Login crea sesión una vez.
- Refresh rota una vez y conserva family ID.
- Replay revoca familia.
- Dos refresh concurrentes: uno exitoso.
- Token expirado, usuario inactivo, store caído y context cancelado.
- Ningún error o log contiene token plano/hash completo.
- Familia revocada impide nuevas rotaciones.

## Criterios de término

- [ ] Todo refresh emitido tiene una sesión persistida por puerto.
- [ ] El token anterior se consume atómicamente al crear el siguiente.
- [ ] Replay revoca la familia.
- [ ] Solo se almacena SHA-256, nunca token plano.
- [ ] GetToken público ya no permite saltarse el store.
- [ ] Concurrencia, race, suite y vet pasan.
- [ ] Contrato del adapter está documentado.
- [ ] Fila 006 marcada `DONE`.

## Condiciones de parada

- El datastore del consumidor no puede garantizar compare-and-swap/transacción.
- El equipo prefiere refresh puramente stateless y acepta explícitamente replay.
- Se requiere compatibilidad con refresh tokens antiguos sin family ID.
- El plan 003 no ofrece jti criptográficamente aleatorio.

## Notas de mantenimiento

La limpieza de sesiones expiradas pertenece al adapter. Revisar métricas de
replay y fallos de rotación sin registrar tokens. Cambiar el hash o el esquema
requiere una estrategia de migración del store.
