package db

import _ "embed"

//go:embed schema.sql
var Schema string

type ResourceType int64

const (
	MOODLE_RESOURCE_GENERIC ResourceType = iota
	MOODLE_RESOURCE_FILE
	MOODLE_RESOURCE_BOOK
	MOODLE_RESOURCE_HTML_AREA
)
