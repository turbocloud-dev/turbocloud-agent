package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/rqlite/gorqlite"
)

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hi there, I love %s!", r.URL.Path[1:])
}

func main() {

	conn, _ := gorqlite.Open("http://") // same only explicitly

	statements := make([]string, 0)

	statements = append(statements, "CREATE TABLE secret_agents (id INTEGER NOT NULL PRIMARY KEY, hero_name TEXT, abbrev TEXT)")

	pattern := "INSERT INTO secret_agents(id, hero_name, abbrev) VALUES (%d, '%s', '%3s')"
	statements = append(statements, fmt.Sprintf(pattern, 125718, "Speed Gibson", "Speed"))
	statements = append(statements, fmt.Sprintf(pattern, 209166, "Clint Barlow", "Clint"))
	statements = append(statements, fmt.Sprintf(pattern, 44107, "Barney Dunlap", "Barney"))
	results, _ := conn.Write(statements)

	// now we have an array of []WriteResult

	for n, v := range results {
		fmt.Printf("for result %d, %d rows were affected\n", n, v.RowsAffected)
		if v.Err != nil {
			fmt.Printf("   we have this error: %s\n", v.Err.Error())
		}
	}

	var id int64
	var name string
	rows, _ := conn.QueryOne("select id, hero_name from secret_agents where id > 500")
	fmt.Printf("query returned %d rows\n", rows.NumRows())
	for rows.Next() {
		err := rows.Scan(&id, &name)
		fmt.Println("got " + name)
		if err != nil {
			fmt.Printf(" Cannot run Scan: %s\n", err.Error())
		}
	}

	http.HandleFunc("/hey", handler)
	create_service()

	PORT := "5445"
	fmt.Println("Starting an agent on port " + PORT)
	log.Fatal(http.ListenAndServe(":"+PORT, nil))
}
