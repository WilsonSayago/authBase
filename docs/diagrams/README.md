# Diagramas (Archify)

Diagramas interactivos de la librería `authBase` v3. Abrí los `.html` en el navegador (tema claro/oscuro, pan/zoom, vistas guiadas).

> Contenido autorado en español. La UI fija del visor Archify permanece en inglés (`html lang` fallback).

| Diagrama | Tipo | HTML | Especificación |
|----------|------|------|----------------|
| Mapa de componentes | architecture | [authbase-architecture.html](./authbase-architecture.html) | [JSON](./authbase-architecture.json) |
| Login y refresh | workflow | [authbase-login-refresh.html](./authbase-login-refresh.html) | [JSON](./authbase-login-refresh.workflow.json) |
| Ciclo de sesión refresh | lifecycle | [authbase-refresh-lifecycle.html](./authbase-refresh-lifecycle.html) | [JSON](./authbase-refresh.lifecycle.json) |

## Regenerar

Desde el paquete Archify instalado en el entorno del agente:

```sh
ARCHIFY="$HOME/.agents/skills/archify/bin/archify.mjs"
REPO="$(pwd)"  # raíz de authBase
DIR=docs/diagrams

node "$ARCHIFY" deliver architecture "$DIR/authbase-architecture.json" \
  "$DIR/authbase-architecture.html" --quality showcase --repo-root "$REPO"

node "$ARCHIFY" deliver workflow "$DIR/authbase-login-refresh.workflow.json" \
  "$DIR/authbase-login-refresh.html" --quality showcase

node "$ARCHIFY" deliver lifecycle "$DIR/authbase-refresh.lifecycle.json" \
  "$DIR/authbase-refresh-lifecycle.html" --quality showcase
```
