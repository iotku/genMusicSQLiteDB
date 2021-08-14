package main

import (
	"database/sql"
	"fmt"
	"log"
)

// InitDB creates a new sqlite database with provided table containing provided columns
// Note: path is hardcoded to exist so shouldn't be included in columns
func InitDB(dbfile string, table string, columns ...string) (*sql.Tx, *sql.DB) {
	if len(columns) < 1 {
		log.Fatalln("InitDB() requires at least one column")
	}
	var db *sql.DB
	db, err := sql.Open("sqlite3", dbfile)

	ckErrFatal(err)

	ddl := `
	       PRAGMA automatic_index = ON;
	       PRAGMA cache_size = 32768;
	       PRAGMA cache_spill = OFF;
	       PRAGMA foreign_keys = ON;
	       PRAGMA journal_size_limit = 67110000;
	       PRAGMA locking_mode = NORMAL;
	       PRAGMA page_size = 4096;
	       PRAGMA recursive_triggers = ON;
	       PRAGMA secure_delete = OFF;
	       PRAGMA synchronous = OFF;
	       PRAGMA temp_store = MEMORY;
	       PRAGMA journal_mode = OFF;
	       PRAGMA wal_autocheckpoint = 16384;
	       CREATE TABLE IF NOT EXISTS `
	ddl += "\"" + table + "\" ("
	for _, col := range columns {
		ddl += "           \"" + col + "\" TEXT NOT NULL,"
	}
	ddl += "\"path\" TEXT NOT NULL);"
    ddl += "CREATE UNIQUE INDEX IF NOT EXISTS \"path\" ON \"" + table + "\" (\"path\");"

	_, err = db.Exec(ddl)
	if err != nil {
		log.Fatal(err)
	}

	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}

	return tx, db
}

// Open a SQLite3 Database
// Returns a *sql.DB struct
func openDB(dbfile string) *sql.DB {
	database, err := sql.Open("sqlite3", dbfile)
	ckErrFatal(err)
	return database
}

// TODO: Consider filtering non alphabetical chars to avoid SQLi here (;?), although we assume trusted input
func genInsertStr(table string, columns ...string) string {
	if len(columns) < 1 {
		log.Fatalln("genInsertStr() requires at least one column")
	}
	outstring := "INSERT into \"" + table + "\" ("
	for _, value := range columns {
		outstring += value + ", "
	}
	outstring = outstring[0:len(outstring)-2] + ") VALUES ("
	for i := 0; i < len(columns); i++ {
		outstring += "?, "
	}
	return outstring[0:len(outstring)-2] + ");"
}

func PrepareStatementInsert(tx *sql.Tx, table string, columns ...string) *sql.Stmt {
	stmt, err := tx.Prepare(genInsertStr(table, columns...))
	if err != nil {
		log.Fatal(err)
	}
	return stmt
}

func PrepareStatementRemove(tx *sql.Tx, table string) *sql.Stmt {
	stmt, err := tx.Prepare("DELETE FROM " + table + " WHERE path = ?")
	if err != nil {
		log.Fatal(err)
	}
	return stmt
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

// GetRowCount returns amount of rows in a table
func GetRowCount(database *sql.DB, table string) (count uint64) {
	ckErrFatal(database.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count))
	return
}