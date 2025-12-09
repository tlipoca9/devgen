package common

// Status is an enum type in common package for testing cross-package references.
// enumgen:@enum(string)
type Status int

const (
	StatusActive Status = iota + 1
	StatusInactive
	StatusPending
)

// Priority is a numeric type for testing cross-package field types.
type Priority int

const (
	PriorityLow    Priority = 1
	PriorityMedium Priority = 2
	PriorityHigh   Priority = 3
)

// Level is a string type for testing cross-package field types.
type Level string

const (
	LevelDebug Level = "debug"
	LevelInfo  Level = "info"
	LevelWarn  Level = "warn"
	LevelError Level = "error"
)
