package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

var dbfile = "./media.db"
var rootDir string
var prefix string
var trimStr string
var dirAmnt int

func main() {
	prefixPtr := flag.String("prefix", "", "Add a prefix to the path (For example, if you would rather have /Music -> /nas/Music you would use -prefix=/nas")
	trimPtr := flag.String("trim", "", "Trim the provided characters from the beginning of the path. (For example, if you wanted to change /mnt/Music -> /Music you would provide -trim=/mnt)")
	flag.Parse()
	if len(flag.Args()) < 1 {
		showHelp()
		log.Fatalln("You must provide a path, exiting.")
	}
	rootDir = flag.Args()[0]
	fmt.Println("Prefix: " + *prefixPtr)
	prefix = *prefixPtr

	fmt.Println("Trim: " + *trimPtr)
	if strings.HasPrefix(rootDir, *trimPtr) {
		trimStr = *trimPtr
	} else {
		log.Fatalln("Invalid trim argument, must match beginning of path \"" + rootDir + "\"")
	}

	// Determine how many characters to trim from new prefix to assist in getting original path
	dirAmnt = len(addPrefixAndTrim(rootDir))

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

func addPrefixAndTrim(filePath string) string {
	return prefix + strings.TrimPrefix(filePath, trimStr)
}

func getOriginalFile(path string) string {
	return rootDir + path[dirAmnt:]
}

func showHelp() {
	fmt.Println("genMusicSQLDB vDev")
	fmt.Println("Usage: genMusicSQLiteDB [-option=value] directory")
	fmt.Println("Options")
	flag.PrintDefaults()
}
