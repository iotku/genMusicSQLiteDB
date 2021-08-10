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

	var sqltx *sql.Tx
	var database *sql.DB // Closed by scan/compare functions, I think. (unclear, but seems functional)

	if _, err := os.Stat(dbfile); os.IsNotExist(err) {
		// File doesn't exist, so do full DB run without comparison
		fmt.Println("Generate DB")
		sqltx, database = InitDB(dbfile, "music", "artist", "album", "title")
		fullScan(path, sqltx)
	} else {
		database = openDB(dbfile)
		var count int
		err = database.QueryRow("SELECT COUNT(*) FROM music").Scan(&count)
		checkErr(err)
		fmt.Printf("DB Has %d Rows\n", count)
		sqltx, err = database.Begin()
		checkErr(err)
		if count == 0 {
			// Run full scan without checking database
			fullScan(path, sqltx)
		} else {
			// Run Comparison against recently added files
			compareDatabase(path, database, sqltx)
		}
	}

	err := sqltx.Commit()
	if err != nil {
		log.Fatal(err)
	}
	return
}

func compareDatabase(path string, database *sql.DB, tx *sql.Tx) {
	stmt := PrepareStatementInsert(tx)
	defer stmt.Close()

	// Is there a good way to do the comparison during the scan?
	currentFiles := scanDir(path)

	var previousFiles []string
	previousFiles = loadOldFilesList(database)

	// Add these files
	filesToAdd := difference(currentFiles, previousFiles)

	for _, file := range filesToAdd {
		tags, err := getTags(file)
		if tags == nil {
			errorednum++
			printStatus("Error", err.Error()+" "+file)
			continue
		}
		addPathToDB(tags, stmt)
	}

	// remove these
	stmt = PrepareStatementRemove(tx)
	filesToRemove := difference(previousFiles, currentFiles)

	for _, file := range filesToRemove {
		fmt.Println(file)
		removePathFromDB(file, stmt)
	}

	fmt.Printf("\n%d:%d\n", len(currentFiles), len(previousFiles))
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

func loadOldFilesList(database *sql.DB) []string {
	var files []string

	rows, err := database.Query("SELECT path FROM music")
	//defer rows.Close()
	checkErr(err)

	var path string
	for rows.Next() {
		err := rows.Scan(&path)
		files = append(files, path)
		checkErr(err)
	}
	return files
}

func addPathToDB(metadata map[string]string, stmt *sql.Stmt) {
	_, err := stmt.Exec(metadata["artist"], metadata["album"], metadata["title"], metadata["path"])
	if err != nil {
		// Early return if INSERT fails (hopefully because path already exists)
		log.Println(err)
		return
	}
	processednum++
	printStatus("Added", metadata["path"])
}

func removePathFromDB(path string, stmt *sql.Stmt) {
	_, err := stmt.Exec(path)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	removednum++
	printStatus("Removed", path)
}

func printStatus(action, path string) {
	fmt.Printf("Added: %d Error: %d Removed: %d | %s: %s\n", processednum, errorednum, removednum, action, path)
}
