# genMusicSQLiteDB
Generate a SQLite Database of Music files (and accociated metadata) from a folder

genMusicSQLiteDB requires go 1.16 or later as it implements [filepath.WalkDir](https://pkg.go.dev/path/filepath#WalkDir).

Tags are read via the "tag" library: https://github.com/dhowden/tag

sqlite3 driver: https://github.com/mattn/go-sqlite3 (requires gcc)

## Usage

### Create media.db for mumzic
`$ genMusicSQLiteDB [path/to/music/directory]`

This creates a media.db (Currently hardcoded filename) containing 4 colums ("artist", "album", "title", "path")

### Supported Formats
Currently genMusicSQLiteDB program looks for .flac & .mp3 files. Thusfar we have no ability to scan tags for opus files.
