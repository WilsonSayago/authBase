# Plan 003: Centralizar y endurecer JWT y configuración

> **Instrucciones para el ejecutor**: lee el plan completo. Ejecuta todas las
> verificaciones. Si la continuidad de tokens existentes es un requisito, para
> y repórtalo: este cambio define un contrato JWT nuevo para v3.
>
> **Drift check inicial**:
> `git diff --stat e43e3ac..HEAD -- core/authenticationusecase.go core/services/authenticationservice.go core/services/authorizationservice.go infra/config/properties/jwt.go`
> Cambios esperados: tests de 001 y constructores simples de 002. Otros cambios
> de firmas o claims son condición de parada.

## Estado

- **Prioridad**: P1
- **Esfuerzo**: L
- **Riesgo**: HIGH
- **Depende de**: planes 001 y 002
- **Categoría**: security, tech-debt
- **Planificado en**: commit `e43e3ac`, 2026-09-17

## Por qué importa

La librería duplica parsing JWT, acepta cualquier algoritmo HMAC, no exige
expiración, no distingue access de refresh y hace assertions de claims que
pueden provocar panic. La configuración solo valida una clave y llama
`log.Fatal`, terminando el proceso consumidor. Este plan crea un único
TokenManager tipado, validado y testeable.

## Estado actual

- `authenticationservice.go:31-34` define `MyCustomClaims` con `User` y
  `RegisteredClaims`.
- `authenticationservice.go:36-74` firma access y refresh con HS256, pero usa el
  ID de usuario también como `jti`.
- `authenticationservice.go:98-140` usa MapClaims y assertions directas.
- `authorizationservice.go:39-44` repite el parser.
- `jwt.go:22-25` valida solo access secret mediante `log.Fatal`.
- `authenticationusecase.go:10` expone `ValidateTokenAndRefresh`, cuya
  implementación en `authenticationservice.go:143-145` hace panic siempre.

La biblioteca `golang-jwt/jwt/v5` ofrece `WithValidMethods`,
`WithExpirationRequired`, `WithIssuer`, `WithAudience`, `WithIssuedAt`,
`WithLeeway` y `WithStrictDecoding`; usarlas en vez de validación manual.

## Comandos

| Propósito | Comando | Resultado esperado |
|---|---|---|
| Tests token | `env GOCACHE=/private/tmp/authbase-go-cache GOWORK=off go test ./core/services -run 'TestToken|TestAuthentication' -count=1` | exit 0 |
| Race | `env GOCACHE=/private/tmp/authbase-go-cache GOWORK=off go test -race ./...` | exit 0 |
| Vet | `env GOCACHE=/private/tmp/authbase-go-cache GOWORK=off go vet ./...` | exit 0 |
| Panics/TODO | `rg -n 'panic\(|TODO implement|MapClaims|SigningMethodHMAC' core infra` | sin coincidencias en flujo JWT |

## Alcance

**Dentro del alcance**:

- Crear `core/services/token_manager.go` y `token_manager_test.go`.
- `core/services/authenticationservice.go` y tests.
- `core/authenticationusecase.go`.
- `infra/config/properties/jwt.go` y tests nuevos.
- Cambios mínimos en `authorizationservice.go` para recibir un TokenManager;
  el comportamiento HTTP se completa en 005.
- `plans/README.md`.

**Fuera del alcance**:

- No implementar almacenamiento o rotación de refresh; plan 006.
- No cambiar dependencias; plan 007.
- No cambiar module path; plan 008.
- No conservar compatibilidad silenciosa con tokens sin tipo/issuer/audience.
- No registrar tokens, claims, claves ni errores criptográficos detallados.

## Flujo Git

- Rama: `codex/003-secure-jwt`.
- Commits sugeridos: `security: validate jwt configuration` y
  `security: centralize token handling`.

## Pasos

### Paso 1: transformar la configuración en validación retornable

Ampliar `Jwt` con issuer, audience y leeway documentados en YAML. Cambiar
`Validate` para retornar `error`; debe comprobar:

- access y refresh secrets no vacíos y con mínimo 32 bytes;
- secretos diferentes;
- expiraciones positivas y refresh mayor que access;
- issuer y audience no vacíos;
- leeway no negativo y con máximo razonable documentado.

Eliminar `log.Fatal`. No validar “entropía” mediante heurísticas. Los tests deben
construir secretos sintéticos en memoria sin copiar valores del repositorio.

**Verificar**: `go test ./infra/config/properties -run TestJwtPropValidate -count=1`
→ todos los casos en tabla pasan.

### Paso 2: crear claims y TokenManager tipados

En `token_manager.go`, definir tipos no ambiguos para `access` y `refresh`.
Usar `RegisteredClaims.Subject` como user ID, `ID` como identificador aleatorio
criptográfico y un claim propio para tipo. Generar al menos 128 bits aleatorios
con `crypto/rand` y codificación URL-safe.

TokenManager debe copiar la configuración por valor y permitir inyectar un
reloj y generador de ID privados para tests. La API productiva debe ofrecer:

- emitir par access/refresh;
- parsear exclusivamente access;
- parsear exclusivamente refresh;
- devolver subject y claims tipados, nunca MapClaims.

Parsing: HS256 exacto, expiración obligatoria, issuer/audience obligatorios,
issued-at validado, strict decoding y leeway configurado.

**Verificar**: tests para token correcto, algoritmo distinto, secret incorrecto,
exp ausente, expirado, issuer/audience/tipo incorrectos, subject vacío y claim
malformado → todos pasan sin panic.

### Paso 3: hacer explícito el error de construcción

Crear constructores que retornen `(instancia, error)` cuando la configuración
sea inválida. Guardar TokenManager en AuthenticationService y Authorization en
vez de punteros mutables a JwtProp. No usar panic ni log.Fatal.

Si se mantienen temporalmente nombres legacy, deben estar marcados deprecated y
retornar error; no crear wrappers que ignoren el error.

**Verificar**: tests con config nil/cero/inválida devuelven error y no panic.

### Paso 4: migrar AuthenticationService

Reemplazar generación y parsing directo por TokenManager. Eliminar
`MyCustomClaims`, MapClaims y type assertions. Retirar
`ValidateTokenAndRefresh` de la interfaz y del servicio: es una API sin contrato
y siempre hace panic. El plan 006 definirá la única operación de refresh.

Conservar por ahora la consulta de usuario existente; el estado activo cambia
en 004.

**Verificar**:
`go test ./core/services -run 'TestTokenManager|TestAuthentication' -count=20`
→ exit 0.

### Paso 5: preparar Authorization para el verificador único

Inyectar TokenManager en Authorization y retirar JwtProp. Puede seguir usando el
flujo HTTP actual hasta 005, pero no puede llamar `jwt.Parse` directamente.

**Verificar**:
`rg -n 'jwt\.Parse|jwt\.New|MapClaims' core/services`
→ las únicas operaciones JWT están en `token_manager.go` y sus tests.

## Plan de pruebas

- Tabla completa de validación de configuración.
- Emisión determinista mediante clock/ID inyectados.
- Access no aceptado como refresh y viceversa.
- Algoritmos HS384/HS512 y claims incompletos rechazados.
- Fuzz test opcional `FuzzParseTokenNeverPanics` con semillas válidas e
  inválidas; nunca debe revelar secretos.
- Suite previa completa y race detector.

## Criterios de término

- [ ] Un solo archivo productivo usa la librería JWT directamente.
- [ ] No hay MapClaims, assertions de claims ni `SigningMethodHMAC` genérico.
- [ ] Config inválida retorna error; no hay log.Fatal.
- [ ] `ValidateTokenAndRefresh` y su TODO desaparecieron.
- [ ] Access y refresh tienen tipo, subject, jti aleatorio, issuer, audience,
  iat y exp obligatorios.
- [ ] Tests hostiles y `go test -race ./...` pasan.
- [ ] Fila 003 marcada `DONE`.

## Condiciones de parada

- Algún consumidor exige que tokens emitidos antes del cambio sigan válidos.
- Se solicita aceptar múltiples algoritmos o claves sin una política de key ID.
- Algún consumidor exige acoplar la carga de configuración JWT a initModules en
  vez de recibir configuración validada por constructor.
- El cambio requiere persistencia de refresh antes del plan 006.

## Notas de mantenimiento

Este plan cambia el wire format JWT y debe desplegarse coordinadamente con los
consumidores. Revisar especialmente que logs y errores no contengan tokens ni
claves. La futura rotación de claves debe diseñarse con `kid`, no relajando
`WithValidMethods`.
