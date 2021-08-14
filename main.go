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
var rootDir string

func main() {
	flag.Parse()
	rootDir = flag.Arg(0)

	var sqlTx *sql.Tx
	var database *sql.DB // Closed by scan/compare functions, I think. (unclear, but seems functional)

	if _, err := os.Stat(dbfile); os.IsNotExist(err) {
		// File doesn't exist, so do full DB run without comparison
		fmt.Println("Generate DB")
		sqlTx, database = InitDB(dbfile, "music", "artist", "album", "title")
		fullScan(rootDir, sqlTx)
	} else {
		database = openDB(dbfile)

		count := GetRowCount(database, "music")
		fmt.Printf("DB Has %d Rows\n", count)

		sqlTx, err = database.Begin()
		ckErrFatal(err)

		if count == 0 {
			// Run full scan without checking database
			fullScan(rootDir, sqlTx)
		} else {
			// Run Comparison against recently added files
			compareDatabase(rootDir, database, sqlTx)
		}
	}

	ckErrFatal(sqlTx.Commit())
	return
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

// Log an error and terminate
func ckErrFatal(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}