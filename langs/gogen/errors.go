package gogen

import "errors"

// ErrGenerateGoCode is returned when Go code generation encounters unrecoverable metadata issues.
var ErrGenerateGoCode = errors.New("gogen: generate go code failure")
