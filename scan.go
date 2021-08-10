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
	stmt := PrepareStatementInsert(tx)
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

func isValidExt(ext string) bool {
	// Tag doesn't currently work with opus files.
	// https://github.com/dhowden/tag/pull/69
	return ext == ".flac" || ext == ".mp3"
}