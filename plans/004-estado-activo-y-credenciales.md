# Plan 004: Aplicar estado activo y aislar credenciales

> **Instrucciones para el ejecutor**: ejecutar después del TokenManager. Este
> plan cambia puertos públicos; si existe un consumidor que no puede migrar a
> context.Context o separar credenciales, detenerse y enumerarlo.
>
> **Drift check inicial**:
> `git diff --stat e43e3ac..HEAD -- core/domain core/port core/services infra/secundary`
> Esperar tests, eliminación de singletons y TokenManager. Comparar todas las
> firmas antes de actuar.

## Estado

- **Prioridad**: P1
- **Esfuerzo**: L
- **Riesgo**: HIGH
- **Depende de**: `plans/003-endurecer-jwt-y-configuracion.md`
- **Categoría**: security, tech-debt
- **Planificado en**: commit `e43e3ac`, 2026-09-17

## Por qué importa

`Active` existe pero no participa en autenticación ni autorización. Un usuario
desactivado puede iniciar sesión o refrescar, y un rol inactivo sigue aportando
permisos. Además, `IUserGeneric` expone el hash de contraseña y UserGeneric
ofrece una comparación directa de strings, ampliando innecesariamente la
superficie de credenciales.

## Estado actual

- `core/domain/userGeneric.go:3-10` exige `GetPassword` pero no `GetActive`.
- `userGeneric.go:50-71` agrega roles sin mirar `role.Active`.
- `userGeneric.go:96-99` compara password mediante `==`.
- `authenticationservice.go:77-95` distingue usuario ausente de password
  incorrecto y no revisa estado.
- `RefreshToken` actualmente no recarga al usuario antes de emitir tokens.
- `GenericPort` mezcla búsqueda de credenciales con lectura de perfil.

## Comandos

| Propósito | Comando | Resultado esperado |
|---|---|---|
| Dominio | `go test ./core/domain -run 'Test.*Active|Test.*Permission' -count=20` | exit 0 |
| Servicios | `go test ./core/services -run 'Test(Login|Refresh|Validate).*Active|TestInvalidCredentials' -count=20` | exit 0 |
| Búsqueda | `rg -n 'GetPassword|CheckPassword\(' core/domain/userGeneric.go` | sin salida |
| Race | `go test -race ./...` | exit 0 |

Ejecutar comandos Go con `GOCACHE=/private/tmp/authbase-go-cache GOWORK=off`.

## Alcance

**Dentro del alcance**:

- `core/domain/userGeneric.go`, `role.go` y tests.
- Refactor de `core/port/genericport.go` en puertos separados de credenciales y
  lectura de usuario; crear archivos con nombres claros si es preferible.
- `core/services/authenticationservice.go`, `authorizationservice.go` y tests.
- `core/authenticationusecase.go` si necesita context.Context.
- `core/port/validationport.go` y ValidationService solo para preservar hashing
  y verificación bcrypt fuera del dominio.
- Errores sentinel en un archivo nuevo `core/errors.go`.

**Fuera del alcance**:

- No cambiar formato JWT; ya quedó definido en 003.
- No diseñar refresh replay; 006.
- No añadir un ORM o repositorio concreto.
- No registrar si el usuario existe o por qué falló una credencial.
- No exponer hashes en errores, DTOs o logs.

## Flujo Git

- Rama: `codex/004-account-state`.
- Commits: `security: enforce active identities` y
  `refactor: isolate credential records`.

## Pasos

### Paso 1: volver explícito el estado de identidad

Añadir `GetActive() bool` a IUserGeneric e implementarlo. En
`GetPermissions`, ignorar roles con `Active == false`. Mantener la unión OR para
roles activos.

Tests obligatorios:

- usuario activo/inactivo;
- rol activo concede;
- rol inactivo no concede;
- mezcla de rol activo e inactivo;
- admin inactivo nunca debe considerarse autorizado en servicios.

**Verificar**: tests de dominio pasan 20 veces.

### Paso 2: separar perfil y credenciales

Reemplazar GenericPort por contratos explícitos:

- un lector de usuarios completos por ID para validación/autorización;
- un lector de credenciales por username/email que retorne usuario y hash en un
  record dedicado, no mediante IUserGeneric.

Agregar `context.Context` a operaciones de puertos que puedan tocar I/O. Retirar
`GetPassword` de IUserGeneric y eliminar `UserGeneric.CheckPassword`. Mantener
ValidationPort como verificador/hash bcrypt inyectable, o renombrarlo a
PasswordHasher si la compatibilidad ya no importa.

No mezclar error de “no encontrado” con un error de infraestructura: definir
sentinels internos para que el servicio pueda devolver `ErrInvalidCredentials`
al caller sin perder la causa operativa mediante wrapping.

**Verificar**: `rg` no encuentra exposición de password en el dominio y todos
los fakes compilan con context.

### Paso 3: aplicar política de login uniforme

Login debe devolver el mismo error público para usuario inexistente, password
incorrecto o usuario inactivo. No debe emitir tokens en ninguno. Errores reales
del repositorio deben seguir siendo distinguibles mediante `errors.Is` para que
la aplicación pueda responder 500 sin enumerar cuentas.

**Verificar**: tests comparan `errors.Is`, no strings.

### Paso 4: revalidar usuario en access y refresh

Después de parsear subject:

- `ValidateToken` obtiene el usuario, exige existencia y `Active`;
- refresh obtiene el usuario de nuevo, exige `Active` y solo entonces emite;
- Authorization aplica la misma regla antes de guardar en contexto.

No confiar en estado incluido dentro del JWT.

**Verificar**: un usuario desactivado después de emitir el token no puede validar
ni refrescar.

## Plan de pruebas

- Estado activo en dominio, login, validate y refresh.
- Roles inactivos y admin inactivo.
- Errores de credenciales indistinguibles externamente.
- Errores de DB todavía detectables por `errors.Is`.
- Fakes sensibles a context cancelado.
- Suite completa con race detector.

## Criterios de término

- [ ] IUserGeneric no expone contraseña y sí expone estado activo.
- [ ] No existe comparación directa de contraseña.
- [ ] Puertos de credenciales y lectura están separados y reciben context.
- [ ] Usuario inactivo falla en login, validate, refresh y authorization.
- [ ] Rol inactivo no concede permisos.
- [ ] Tests y vet pasan; no se comparan mensajes de error frágiles.
- [ ] Fila 004 marcada `DONE`.

## Condiciones de parada

- Un consumidor exige obtener el hash desde IUserGeneric.
- No puede definirse cómo distinguir “not found” de error de infraestructura.
- La migración a context.Context requiere cambiar repositorios fuera del alcance
  sin que el operador los haya puesto disponibles.
- La semántica de `Role.Active` no significa “rol deshabilitado”.

## Notas de mantenimiento

El puerto de credenciales es una frontera sensible: revisar que ningún DTO o
log exponga el hash. Los consumidores deben mapear sus errores de persistencia a
los sentinels documentados por authBase.
