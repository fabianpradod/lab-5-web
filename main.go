package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"log"
	"net"
	"strings"

	_ "modernc.org/sqlite"
)

func main() {
	db, err := sql.Open("sqlite", "file:tv-shows.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	db.Exec("CREATE TABLE IF NOT EXISTS shows (title TEXT PRIMARY KEY, current_episode TEXT)")

	shows := [][]string{
		{"A Knight of the Seven Kingdoms", "S01E05"},
		{"Bojack Horseman", "S06E15"},
	}

	for _, s := range shows {
		db.Exec("INSERT OR IGNORE INTO shows (title, current_episode) VALUES (?, ?)", s[0], s[1])
	}

	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()

	log.Print("listening to port 8080")

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

	requestLine, err := reader.ReadString('\n')
	if err != nil {
		return
	}

	parts := strings.Fields(requestLine)
	if len(parts) < 2 {
		return
	}

	path := parts[1]

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		if line == "\r\n" {
			break
		}
	}

	var body string
	if path == "/shows" {
		rows, err := db.Query("SELECT title, current_episode FROM shows ORDER BY title")
		if err != nil {
			body = "<html><body><h1>Error querying</h1></body></html>"
		} else {
			defer rows.Close()
			var b strings.Builder
			b.WriteString("<html><body><h1>Shows</h1><table border=\"1\" cellpadding=\"6\" cellspacing=\"0\">")
			b.WriteString("<tr><th>Title</th><th>Current Episode</th></tr>")

			for rows.Next() {
				var title, ep string
				if err := rows.Scan(&title, &ep); err != nil {
					continue
				}
				b.WriteString(fmt.Sprintf("<tr><td>%s</td><td>%s</td></tr>", title, ep))
			}

			b.WriteString("</table></body></html>")
			b.WriteString("<script>alert('Shows loaded successfully!');</script>")
			body = b.String()
		}
	} else {
		body = "<html><body><h1>SQLite Shows Server</h1><p>Visit <a href=\"/shows\">/shows</a></p></body></html>"
	}

	resp := fmt.Sprintf(
		"HTTP/1.1 200 OK\r\n"+
			"Content-Type: text/html; charset=utf-8\r\n"+
			"Content-Length: %d\r\n"+
			"Connection: close\r\n"+
			"\r\n"+
			"%s",
		len(body),
		body,
	)

	conn.Write([]byte(resp))
}
