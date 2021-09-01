package main

import (
	"database/sql"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"

	"github.com/dhowden/tag"
)

var processednum = 0
var errorednum = 0
var removednum = 0

func fullScan(rootDir string, tx *sql.Tx) {
	stmt := PrepareStatementInsert(tx, "music", "artist", "album", "title", "path")
	defer stmt.Close()

	err := filepath.WalkDir(rootDir, func(path string, info fs.DirEntry, err error) error {
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
// Notably this does not read tags, but rather produces a file list for comparisons.
func scanPath(path string) []string {
	var fileList []string
	err := filepath.WalkDir(path, func(path string, info fs.DirEntry, err error) error {
		if isValidExt(filepath.Ext(path)) {
			fileList = append(fileList, addPrefixAndTrim(path))
		}
		return nil
	})

	if err != nil {
		fmt.Println(err.Error())
	}
	return fileList
}

func getTags(filePath string) (map[string]string, error) {
	// Clean the filepath to make security scanner happy.
	// Personally, I don't think this really matters given our use case as we don't accept arbitrary input remotely,
	// however this may be a good idea regardless in case the scope changes at some point.
	// TODO: Investigate if performance impact is non-negligible
	//filePath = filepath.Clean(filePath)

	// Verify that our prefix has not changed after filepath has been cleaned
	//	if !strings.HasPrefix(filePath, rootDir) {
	//		panic(fmt.Errorf("getTags() Invalid path prefix at '" + filePath + "'doesn't have prefix '" + rootDir + "'"))
	//	}

	f, err := os.Open(filePath)
	if err != nil {
		log.Println("Tag couldn't open filePath:", filePath)
		return nil, err
	}

	meta, err := tag.ReadFrom(f)
	if err != nil {
		ckErrFatal(f.Close())
		return nil, err
	}
	ckErrFatal(f.Close())
	filePath = addPrefixAndTrim(filePath)

	// TODO: Maybe consider just using a standard slice instead?
	metadata := map[string]string{
		"artist": meta.Artist(),
		"album":  meta.Album(),
		"title":  meta.Title(),
		"path":   filePath,
	}
	return metadata, nil
}

func compareDatabase(path string, database *sql.DB, tx *sql.Tx) {
	stmt := PrepareStatementInsert(tx, "music", "artist", "album", "title", "path")
	defer stmt.Close()

	// Is there a good way to do the comparison during the scan?
	currentFiles := scanPath(path)

	var previousFiles []string
	previousFiles = loadOldFilesList(database)

	// Add these files
	filesToAdd := difference(currentFiles, previousFiles)

	for _, file := range filesToAdd {
		file = getOriginalFile(file)
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
	ckErrFatal(err)

	var path string
	for rows.Next() {
		err := rows.Scan(&path)
		files = append(files, path)
		ckErrFatal(err)
	}
	return files
}

func isValidExt(ext string) bool {
	// Tag doesn't currently work with opus files.
	// https://github.com/dhowden/tag/pull/69
	return ext == ".flac" || ext == ".mp3"
}
