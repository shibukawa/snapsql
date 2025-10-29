package schema

import _ "embed"

// SchemaSQL holds the notification example schema.
//
//go:embed schema.sql
var SchemaSQL string
