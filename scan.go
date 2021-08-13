package main

import (
	"database/sql"
	"fmt"
	"github.com/dhowden/tag"
	"io/fs"
	"log"
	"os"
	"path/filepath"
)


var processednum = 0
var errorednum = 0
var removednum = 0

func fullScan(path string, tx *sql.Tx) {
	stmt := PrepareStatementInsert(tx, "music","artist", "album", "title", "path")
	defer stmt.Close()

	err := filepath.WalkDir(path, func(path string, info fs.DirEntry, err error) error {
		if isValidExt(filepath.Ext(path)) {
			tags, err := getTags(path)
			if tags == nil {
				errorednum++
				printStatus("Error", err.Error()+" "+path)
				return nil
			} else {
				addPathToDB(tags, stmt)
			}
		}
		return nil
	})

	if err != nil {
		fmt.Println(err.Error())
	}
}

// Recursively scan path for files to be added or compared to database
func scanDir(path string) []string {
	var fileList []string
	err := filepath.WalkDir(path, func(path string, info fs.DirEntry, err error) error {
		if isValidExt(filepath.Ext(path)) {
			fileList = append(fileList, path)
		}
		return nil
	})

	if err != nil {
		fmt.Println(err.Error())
	}
	return fileList
}

func getTags(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		log.Println("Tag couldn't open path:", path)
		return nil, err
	}
	m, err := tag.ReadFrom(f)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	metadata := map[string]string{
		"artist": m.Artist(),
		"album":  m.Album(),
		"title":  m.Title(),
		"path":   path,
	}

	return metadata, nil
}

func compareDatabase(path string, database *sql.DB, tx *sql.Tx) {
	stmt := PrepareStatementInsert(tx, "music", "artist", "album", "title", "path")
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
	stmt = PrepareStatementRemove(tx, "music")
	filesToRemove := difference(previousFiles, currentFiles)

	for _, file := range filesToRemove {
		fmt.Println(file)
		removePathFromDB(file, stmt)
	}

	fmt.Printf("\n%d:%d\n", len(currentFiles), len(previousFiles))
}

func loadOldFilesList(database *sql.DB) []string {
	var files []string

	rows, err := database.Query("SELECT path FROM music")
	//defer rows.Close()
	checkErrFatal(err)

	var path string
	for rows.Next() {
		err := rows.Scan(&path)
		files = append(files, path)
		checkErrFatal(err)
	}
	return files
}

func isValidExt(ext string) bool {
	// Tag doesn't currently work with opus files.
	// https://github.com/dhowden/tag/pull/69
	return ext == ".flac" || ext == ".mp3"
}