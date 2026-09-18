# Planes de implementación de authBase

Generados con la skill `improve` el 2026-09-17 y reconciliados el mismo día
después de la publicación válida de `initModules/v2`. Ejecutar en el orden
indicado, salvo que las dependencias de un plan indiquen otra cosa. Cada agente
debe leer su plan completo, respetar las condiciones de parada y actualizar su
fila al terminar.

## Contexto común

- Repositorio: `github.com/WilsonSayago/authBase`.
- Commit auditado: `e43e3ac`.
- El árbol de trabajo tiene cambios del usuario en `go.mod` y `go.sum`: cambia
  la directiva Go de `1.24.0` a `1.26.3` y añade
  `github.com/WilsonSayago/initModules/v2 v2.0.0`. También existen cambios o
  archivos `.DS_Store`. Ningún ejecutor debe descartarlos ni incluirlos por
  accidente en commits ajenos a esa migración.
- Durante la redacción apareció un `go.work` no versionado con varios módulos
  locales. También pertenece al usuario: no modificarlo ni incluirlo. Todos los
  gates de estos planes usan `GOWORK=off` para verificar authBase aisladamente.
- La verificación original en `e43e3ac` pasaba sin tests. El estado reconciliado
  ya no compila con `GOWORK=off`: cuatro archivos todavía importan
  `github.com/WilsonSayago/initModules` mientras `go.mod` requiere la ruta
  `/v2`. El plan 001 completa esa migración antes de crear la línea base.
- `initModules/v2 v2.0.0` sí es una dependencia válida. Su `go.mod` declara
  `module github.com/WilsonSayago/initModules/v2` y el tag resuelve sin
  `replace`. La recomendación posterior de retirarla de authBase responde al
  ciclo de vida global de los servicios, no a un problema de versionado.
- Las etiquetas `authBase v2.0.x` tampoco son módulos válidos. La publicación
  corregida se prepara como `v3`, solo al final de esta secuencia.

## Orden y estado

| Plan | Título | Prioridad | Esfuerzo | Depende de | Estado |
|------|--------|-----------|----------|------------|--------|
| 001 | Completar initModules/v2 y establecer pruebas/CI | P1 | M | — | DONE |
| 002 | Eliminar singletons globales e initModules | P1 | M | 001 | TODO |
| 003 | Centralizar y endurecer JWT y configuración | P1 | L | 001, 002 | TODO |
| 004 | Aplicar estado activo y aislar credenciales | P1 | L | 003 | TODO |
| 005 | Hacer el middleware de autorización panic-safe | P1 | M | 003, 004 | TODO |
| 006 | Implementar rotación y detección de replay de refresh tokens | P2 | L | 003, 004 | TODO |
| 007 | Actualizar Go, dependencias y análisis de vulnerabilidades | P1 | S | 002–006 | TODO |
| 008 | Preparar la API y publicación válida de authBase v3 | P2 | L | 001–007 | TODO |

Valores permitidos: `TODO`, `IN PROGRESS`, `DONE`, `BLOCKED: <motivo>` o
`REJECTED: <motivo>`.

## Dependencias

- 001 termina el cambio de imports a `/v2`, restaura una compilación aislada y
  crea los fakes y la red de seguridad que necesitan todos los refactors.
- 002 elimina el ciclo de vida singleton de la librería antes de rediseñar
  constructores; v2 ya corrigió la sincronización entre claves, pero no el
  acoplamiento global ni las colisiones de una misma clave genérica.
- 003 crea el único componente responsable de emitir y validar JWT.
- 004 cambia los puertos de usuario/credenciales y hace que usuarios o roles
  inactivos fallen de forma uniforme.
- 005 consume el verificador tipado de 003 y los puertos de 004.
- 006 cambia el contrato de refresh; debe aterrizar antes de congelar la API v3.
- 007 se ejecuta después de estabilizar imports para que `go mod tidy` produzca
  el grafo final.
- 008 es el único plan autorizado para cambiar el module path; no crea tags ni
  publica releases.

## Hallazgos considerados y no planificados

- Cachear `UserGeneric.GetPermissions`: no vale la pena antes de medir perfiles
  grandes; el costo actual es local y pequeño.
- Añadir almacenamiento concreto para refresh tokens: la librería debe definir
  el puerto y sus garantías atómicas, no imponer PostgreSQL, MongoDB o Redis.
- Cambiar `bcrypt.DefaultCost`: no existe evidencia de que sea el cuello de
  botella. Mantenerlo configurable puede evaluarse después de v3.
- Renombrar `infra/secundary`: es una mejora cosmética con alto costo de imports.
  Puede hacerse en v3 solo si el mantenedor lo selecciona explícitamente.
- Implementar cookies, CORS o CSRF: pertenecen al adaptador HTTP de cada
  consumidor; authBase actualmente opera con tokens Bearer.

## Reglas para todos los ejecutores

- Usar ramas `codex/NNN-<slug>` si se crea una rama.
- No hacer push, tag, release ni PR sin una instrucción explícita del operador.
- No usar `git reset --hard`, `git checkout --` ni borrar cambios preexistentes.
- No copiar valores de secretos desde archivos de configuración a pruebas,
  documentación, logs o planes.
- Antes de cada commit ejecutar como mínimo:

  ```sh
  env GOCACHE=/private/tmp/authbase-go-cache GOWORK=off go test ./...
  env GOCACHE=/private/tmp/authbase-go-cache GOWORK=off go test -race ./...
  env GOCACHE=/private/tmp/authbase-go-cache GOWORK=off go vet ./...
  git diff --check
  ```
