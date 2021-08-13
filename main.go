package main

import (
	"database/sql"
	"flag"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"os"
)

var dbfile = "./media.db"

func main() {
	flag.Parse()
	path := flag.Arg(0)

	var sqlTx *sql.Tx
	var database *sql.DB // Closed by scan/compare functions, I think. (unclear, but seems functional)

	if _, err := os.Stat(dbfile); os.IsNotExist(err) {
		// File doesn't exist, so do full DB run without comparison
		fmt.Println("Generate DB")
		sqlTx, database = InitDB(dbfile, "music", "artist", "album", "title")
		fullScan(path, sqlTx)
	} else {
		database = openDB(dbfile)
		var count int
		err = database.QueryRow("SELECT COUNT(*) FROM music").Scan(&count)
		checkErr(err)
		fmt.Printf("DB Has %d Rows\n", count)
		sqlTx, err = database.Begin()
		checkErr(err)
		if count == 0 {
			// Run full scan without checking database
			fullScan(path, sqlTx)
		} else {
			// Run Comparison against recently added files
			compareDatabase(path, database, sqlTx)
		}
	}

	err := sqlTx.Commit()
	if err != nil {
		log.Fatal(err)
	}
	return
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

// Return the difference between two []string slices, TODO: is there a faster method?
func difference(a, b []string) []string {
	mb := map[string]bool{}
	for _, x := range b {
		mb[x] = true
	}
	ab := []string{}
	for _, x := range a {
		if _, ok := mb[x]; !ok {
			ab = append(ab, x)
		}
	}
	return ab
}

func printStatus(action, path string) {
	fmt.Printf("Added: %d Error: %d Removed: %d | %s: %s\n", processednum, errorednum, removednum, action, path)
}