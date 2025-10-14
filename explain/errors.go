package explain

import "errors"

var (
	// ErrUnsupportedDialect is returned when no collector/parser is available for the specified dialect.
	ErrUnsupportedDialect = errors.New("explain: unsupported dialect")

	// ErrNotImplemented indicates that the requested operation is not yet implemented.
	ErrNotImplemented = errors.New("explain: not implemented")

	// ErrNoDatabase indicates that a CollectorOptions without DB was provided.
	ErrNoDatabase = errors.New("explain: database handle is required")

	// ErrNoPlanRows indicates that the collector executed but did not receive any plan rows.
	ErrNoPlanRows = errors.New("explain: no plan rows returned")
)
