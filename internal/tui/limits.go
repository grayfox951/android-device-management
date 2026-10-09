package tui

import "time"

// defaultTimeout is how long a single adb or fastboot call may take before it
// is considered stuck.
const defaultTimeout = 90 * time.Second

// guardTimeout bounds a Guard, which is normally a single `cat`.
const guardTimeout = 30 * time.Second

// maxLines is how many output lines are kept in memory for display. The full
// log always stays in the temp file.
const maxLines = 3000

// tickInterval is the polling period for a running command.
const tickInterval = 180 * time.Millisecond
