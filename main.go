package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"

	_ "modernc.org/sqlite"
)

func main() {
	db, err := sql.Open("sqlite", "file:tv-shows.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Tabla con id autoincremental para poder hacer UPDATE y DELETE por id
	db.Exec(`CREATE TABLE IF NOT EXISTS shows (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT UNIQUE,
		current_episode INTEGER DEFAULT 1,
		total_episodes INTEGER DEFAULT 1
	)`)

	// Datos de ejemplo
	shows := [][]string{
		{"A Knight of the Seven Kingdoms", "3", "6"},
		{"Bojack Horseman", "15", "77"},
	}
	for _, s := range shows {
		db.Exec("INSERT OR IGNORE INTO shows (title, current_episode, total_episodes) VALUES (?, ?, ?)", s[0], s[1], s[2])
	}

	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()

	log.Print("Escuchando en http://localhost:8080")

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		go handle(conn, db)
	}
}

func handle(conn net.Conn, db *sql.DB) {
	defer conn.Close()

	reader := bufio.NewReader(conn)

	// Leer la primera linea: "GET /ruta HTTP/1.1"
	requestLine, err := reader.ReadString('\n')
	if err != nil {
		return
	}

	parts := strings.Fields(requestLine)
	if len(parts) < 2 {
		return
	}

	method := parts[0]
	fullPath := parts[1]

	// Separar la ruta de los query params: /update?id=3 -> ruta="/update", params="id=3"
	routeParts := strings.SplitN(fullPath, "?", 2)
	route := routeParts[0]
	queryString := ""
	if len(routeParts) > 1 {
		queryString = routeParts[1]
	}

	contentLength := 0
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		if line == "\r\n" {
			break
		}
		// leer el Content-Length para saber cuantos bytes leer del body
		if strings.HasPrefix(line, "Content-Length:") {
			lengthStr := strings.TrimSpace(strings.TrimPrefix(line, "Content-Length:"))
			contentLength, _ = strconv.Atoi(lengthStr)
		}
	}

	body := ""
	if contentLength > 0 {
		buf := make([]byte, contentLength)
		_, err := reader.Read(buf)
		if err != nil {
			return
		}
		body = string(buf)
	}

	// ROUTER — decidir que hacer segun la ruta
	switch {

	// GET / — pagina principal con la tabla de shows
	case method == "GET" && route == "/":
		handleIndex(conn, db)

	// GET /create — formulario para agregar una serie
	case method == "GET" && route == "/create":
		handleCreateForm(conn)

	// POST /create — recibir el formulario y guardar en la DB
	// parsear el body con url.ParseQuery(), hacer INSERT, redirigir con 303
	case method == "POST" && route == "/create":
		handleCreatePost(conn, db, body)

	// POST /update?id=X — incrementar episodio actual
	case method == "POST" && route == "/update":
		handleUpdate(conn, db, queryString)

	// POST /update-prev?id=X — CHALLENGE: boton -1
	case method == "POST" && route == "/update-prev":
		handleUpdatePrev(conn, db, queryString)

	// DELETE /delete?id=X — CHALLENGE: eliminar serie (solo cuenta con DELETE)
	case method == "DELETE" && route == "/delete":
		handleDelete(conn, db, queryString)

	// Servir archivos estaticos: style.css, script.js
	case method == "GET" && strings.HasPrefix(route, "/static/"):
		handleStatic(conn, route)

	// Cualquier otra ruta — 404
	default:
		respond(conn, "HTTP/1.1 404 Not Found", "text/plain", "404 Not Found")
	}
}

// handleIndex — GET / — muestra la tabla con todos los shows
func handleIndex(conn net.Conn, db *sql.DB) {
	rows, err := db.Query("SELECT id, title, current_episode, total_episodes FROM shows ORDER BY title")
	if err != nil {
		respond(conn, "HTTP/1.1 500 Internal Server Error", "text/plain", "Error al consultar la base de datos")
		return
	}
	defer rows.Close()

	var table strings.Builder
	table.WriteString(`
	<table>
		<tr>
			<th>Titulo</th>
			<th>Episodio</th>
			<th>Total</th>
			<th>Progreso</th>
			<th>Acciones</th>
		</tr>`)

	for rows.Next() {
		var id, current, total int
		var title string
		if err := rows.Scan(&id, &title, &current, &total); err != nil {
			continue
		}

		// marcar como completada si current == total
		completedText := ""
		if current >= total {
			completedText = `<span class="completed">Completada</span>`
		}

		// barra de progreso
		percent := 0
		if total > 0 {
			percent = current * 100 / total
		}
		progressBar := fmt.Sprintf(`
			<div class="progress-bar-bg">
				<div class="progress-bar-fill" style="width:%d%%"></div>
			</div> %d%%`, percent, percent)

		table.WriteString(fmt.Sprintf(`
		<tr>
			<td>%s %s</td>
			<td>%d</td>
			<td>%d</td>
			<td>%s</td>
			<td>
				<button onclick="prevEpisode(%d)">-1</button>
				<button onclick="nextEpisode(%d)">+1</button>
				<button onclick="deleteSeries(%d)" style="background:#e74c3c">🗑</button>
			</td>
		</tr>`, title, completedText, current, total, progressBar, id, id, id))
	}
	table.WriteString("</table>")

	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="es">
<head>
	<meta charset="UTF-8">
	<title>Series Tracker Fabian Prado</title>
	<link rel="stylesheet" href="/static/style.css">
	<link rel="icon" href="/static/favicon.ico">
</head>
<body>
	<h1>Series Tracker Fabian Prado</h1>
	<p><a href="/create">+ Agregar nueva serie</a></p>
	%s
	<script src="/static/script.js"></script>
</body>
</html>`, table.String())

	respond(conn, "HTTP/1.1 200 OK", "text/html; charset=utf-8", html)
}

// handleCreateForm — GET /create — muestra el formulario HTML
func handleCreateForm(conn net.Conn) {
	html := `<!DOCTYPE html>
<html lang="es">
<head>
	<meta charset="UTF-8">
	<title>Agregar Serie</title>
	<link rel="stylesheet" href="/static/style.css">
</head>
<body>
	<h1>Agregar Nueva Serie</h1>
	<form method="POST" action="/create">
		<label>Nombre de la serie:</label>
		<input type="text" name="series_name" required>

		<label>Episodio actual:</label>
		<input type="number" name="current_episode" min="1" value="1" required>

		<label>Total de episodios:</label>
		<input type="number" name="total_episodes" min="1" required>

		<button type="submit">Guardar</button>
	</form>
	<br>
	<a href="/">← Volver</a>
</body>
</html>`

	respond(conn, "HTTP/1.1 200 OK", "text/html; charset=utf-8", html)
}

// handleCreatePost — POST /create — lee el body, inserta en BD, redirige
func handleCreatePost(conn net.Conn, db *sql.DB, body string) {
	params := parseFormBody(body)

	name := params["series_name"]
	currentEp := params["current_episode"]
	totalEps := params["total_episodes"]

	// validacion en servidor — verificar que los campos no esten vacios y que current_episode <= total_episodes antes de insertar
	if name == "" || currentEp == "" || totalEps == "" {
		respond(conn, "HTTP/1.1 400 Bad Request", "text/plain", "Faltan campos requeridos")
		return
	}

	_, err := db.Exec(
		"INSERT OR IGNORE INTO shows (title, current_episode, total_episodes) VALUES (?, ?, ?)",
		name, currentEp, totalEps,
	)
	if err != nil {
		respond(conn, "HTTP/1.1 500 Internal Server Error", "text/plain", "Error al guardar")
		return
	}

	// Patron POST/Redirect/GET — despues de insertar, redirigir a /
	redirect(conn, "/")
}

// handleUpdate — POST /update?id=X — suma 1 al episodio actual
func handleUpdate(conn net.Conn, db *sql.DB, queryString string) {
	params := parseQueryString(queryString)
	id := params["id"]

	if id == "" {
		respond(conn, "HTTP/1.1 400 Bad Request", "text/plain", "Falta el id")
		return
	}

	db.Exec(
		"UPDATE shows SET current_episode = current_episode + 1 WHERE id = ? AND current_episode < total_episodes",
		id,
	)

	respond(conn, "HTTP/1.1 200 OK", "text/plain", "ok")
}

// handleUpdatePrev — POST /update-prev?id=X — CHALLENGE: boton -1
func handleUpdatePrev(conn net.Conn, db *sql.DB, queryString string) {
	params := parseQueryString(queryString)
	id := params["id"]

	if id == "" {
		respond(conn, "HTTP/1.1 400 Bad Request", "text/plain", "Falta el id")
		return
	}

	db.Exec(
		"UPDATE shows SET current_episode = current_episode - 1 WHERE id = ? AND current_episode > 1",
		id,
	)

	respond(conn, "HTTP/1.1 200 OK", "text/plain", "ok")
}

// handleDelete — DELETE /delete?id=X — CHALLENGE: eliminar serie
func handleDelete(conn net.Conn, db *sql.DB, queryString string) {
	params := parseQueryString(queryString)
	id := params["id"]

	if id == "" {
		respond(conn, "HTTP/1.1 400 Bad Request", "text/plain", "Falta el id")
		return
	}

	db.Exec("DELETE FROM shows WHERE id = ?", id)

	respond(conn, "HTTP/1.1 200 OK", "text/plain", "ok")
}

func handleStatic(conn net.Conn, route string) {
	fileName := strings.TrimPrefix(route, "/static/")

	var contentType string
	switch {
	case strings.HasSuffix(fileName, ".css"):
		contentType = "text/css"
	case strings.HasSuffix(fileName, ".js"):
		contentType = "application/javascript"
	default:
		respond(conn, "HTTP/1.1 404 Not Found", "text/plain", "Archivo no encontrado")
		return
	}

	data, err := os.ReadFile("static/" + fileName)
	if err != nil {
		respond(conn, "HTTP/1.1 404 Not Found", "text/plain", "Archivo no encontrado")
		return
	}

	respond(conn, "HTTP/1.1 200 OK", contentType, string(data))
}

// UTILS

// parseQueryString — parsea "id=3&foo=bar" -> map{"id":"3", "foo":"bar"}
func parseQueryString(query string) map[string]string {
	result := make(map[string]string)
	if query == "" {
		return result
	}
	pairs := strings.Split(query, "&")
	for _, pair := range pairs {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) == 2 {
			result[kv[0]] = kv[1]
		}
	}
	return result
}

// parseFormBody — parsea el body de un POST con application/x-www-form-urlencoded
func parseFormBody(body string) map[string]string {
	// usar url.ParseQuery(body) para manejar caracteres especiales correctamente
	return parseQueryString(body)
}

// respond — envia una respuesta HTTP completa
func respond(conn net.Conn, status, contentType, body string) {
	response := fmt.Sprintf(
		"%s\r\nContent-Type: %s\r\nContent-Length: %d\r\nConnection: close\r\n\r\n%s",
		status, contentType, len(body), body,
	)
	conn.Write([]byte(response))
}

// redirect — envia una respuesta 303 See Other (patron POST/Redirect/GET)
func redirect(conn net.Conn, location string) {
	response := fmt.Sprintf(
		"HTTP/1.1 303 See Other\r\nLocation: %s\r\nConnection: close\r\n\r\n",
		location,
	)
	conn.Write([]byte(response))
}
