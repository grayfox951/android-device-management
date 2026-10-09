package tui

import "adm/internal/android"

// fileBrowser walks the filesystem starting from the user's home directory and
// can reach any directory above or below it.
type fileBrowser struct {
	viewW   int
	dir     string
	entries []fsEntry
	idx     int
	kind    android.ArgKind
	hidden  bool
	err     error
	// jumping turns on the inline path field.
	jumping bool
}

// fsEntry is one row in the browser.
type fsEntry struct {
	name  string
	path  string
	isDir bool
	up    bool
	size  int64
}
