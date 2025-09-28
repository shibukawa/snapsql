package gogen

import "errors"

var (
	// ErrIteratorRequiresResponseStruct indicates iterator generation was requested without a response struct.
	ErrIteratorRequiresResponseStruct = errors.New("gogen: iterator generation requires a response struct")
	// ErrHierarchicalNodeMissingKeys indicates a hierarchical node definition lacks key fields.
	ErrHierarchicalNodeMissingKeys = errors.New("gogen: hierarchical node has no key fields")
	// ErrHierarchicalMultipleRootKeys indicates multiple root key fields were found where only one is supported.
	ErrHierarchicalMultipleRootKeys = errors.New("gogen: multiple root key fields detected")
)
