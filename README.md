# Lab 5 - Series Tracker
Fabian Prado Dluzniewski #23427

## Challenges implementados

### [Criterio Subjetivo] Estilos y CSS — 10 pts

El archivo `static/style.css` contiene estilos completos para la tabla, formulario, botones, barra de progreso y etiqueta de serie completada. Se usa una paleta coherente, bordes redondeados, sombras y efectos hover.

---

### [Criterio Subjetivo] Codigo de Go ordenado en archivos — 15 pts

Todo el codigo Go esta en `main.go`, organizado en funciones con responsabilidades claras y separadas:

- `main` — inicializa la BD y el listener TCP.
- `handle` — router principal que despacha segun metodo y ruta.
- `handleIndex`, `handleCreateForm`, `handleCreatePost` — logica de vistas.
- `handleUpdate`, `handleUpdatePrev`, `handleDelete` — mutaciones de datos.
- `handleStatic` — servido de archivos estaticos.
- `parseQueryString`, `parseFormBody`, `respond`, `redirect` — utilidades.

---

### [Criterio Subjetivo] Codigo de JavaScript ordenado en archivos — 15 pts

El JavaScript esta separado del HTML en `static/script.js`. El HTML solo incluye `<script src="/static/script.js">`. El archivo define tres funciones async: `nextEpisode`, `prevEpisode` y `deleteSeries`.

---

### Barra de progreso — 15 pts

Implementada en `handleIndex` (`main.go:186`). Por cada serie se calcula el porcentaje de episodios vistos y se genera un elemento `<div class="progress-bar-fill">` con `width` dinamico. El CSS en `style.css` define `.progress-bar-bg` y `.progress-bar-fill`.

---

### Marcar serie completa — 10 pts

En `handleIndex` (`main.go:177`), si `current_episode >= total_episodes` se inserta un `<span class="completed">Completada</span>` junto al titulo de la serie. El estilo `.completed` en el CSS lo resalta en verde y negrita.

---

### Boton -1 — 10 pts

Ruta `POST /update-prev?id=X` manejada por `handleUpdatePrev` (`main.go:302`). La query SQL descuenta un episodio solo si `current_episode > 1`, evitando valores negativos. En el frontend, `prevEpisode(id)` en `script.js` hace el fetch correspondiente.

---

### Funcion para eliminar serie (metodo DELETE) — 20 pts

Ruta `DELETE /delete?id=X` manejada por `handleDelete` (`main.go:320`). El router verifica explicitamente `method == "DELETE"`. En el frontend, `deleteSeries(id)` en `script.js` usa `fetch` con `{ method: "DELETE" }`.

---

### Validacion en servidor de los campos — 25 pts

En `handleCreatePost` (`main.go:265`), antes de ejecutar el INSERT se valida que `series_name`, `current_episode` y `total_episodes` no esten vacios. Si alguno falta, el servidor responde `400 Bad Request` sin tocar la base de datos.

---

### Actualizar sin reload — 20 pts

Las tres acciones del frontend (`nextEpisode`, `prevEpisode`, `deleteSeries`) usan `fetch` de forma asincrona para comunicarse con el servidor sin navegar ni hacer submit de un formulario. Las operaciones de actualizacion de episodios no requieren recarga de pagina completa para enviar el request al servidor.

---

### Servir archivos estaticos — 40 pts

La ruta `GET /static/*` es manejada por `handleStatic` (`main.go:334`), que lee el archivo desde el sistema de archivos con `os.ReadFile` y lo sirve con el `Content-Type` correcto (`.css` -> `text/css`, `.js` -> `application/javascript`). Los archivos `style.css` y `script.js` se sirven de esta forma.

## Estructura del proyecto

```
lab5/
  main.go          <- servidor HTTP + router + handlers + utilidades
  static/
    style.css      <- estilos de la aplicacion
    script.js      <- logica del cliente (fetch async)
  tv-shows.db      <- base de datos SQLite
  go.mod / go.sum  <- modulo Go
```

## Ejecucion

```bash
go run main.go
```

El servidor queda disponible en `http://localhost:8080`.
