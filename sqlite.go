package main

import (
	"database/sql"
	"log"
)

// InitDB creates a new sqlite database with provided table containing provided columns
// Note: path is hardcoded to exist so shouldn't be included in columns
func InitDB(dbfile string, table string, columns ...string) (*sql.Tx, *sql.DB) {
	var db *sql.DB
	db, err := sql.Open("sqlite3", dbfile)

	checkErr(err)

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
	checkErr(err)
	return database
}

// TODO: Consider filtering non alphabetical chars to avoid SQLi here (;?), although we assume trusted input
func genInsertStr(table string, columns ...string) string {
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

func PrepareStatementInsert(tx *sql.Tx) *sql.Stmt {
	stmt, err := tx.Prepare(genInsertStr("music", "artist", "album", "title", "path"))
	if err != nil {
		log.Fatal(err)
	}
	return stmt
}

func PrepareStatementRemove(tx *sql.Tx) *sql.Stmt {
	stmt, err := tx.Prepare(`DELETE FROM music WHERE path = ?`)
	if err != nil {
		log.Fatal(err)
	}
	return stmt
}
