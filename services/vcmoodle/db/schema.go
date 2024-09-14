package db

import _ "embed"

//go:embed schema.sql
var Schema string

type ResourceType int64

const (
	RESOURCE_GENERIC ResourceType = iota
	RESOURCE_FILE
	RESOURCE_BOOK
	RESOURCE_HTML_AREA
)
