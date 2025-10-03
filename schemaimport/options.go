package schemaimport

// Options describes the inputs required to construct a Config instance.
type Options struct {
    // WorkingDir is the base directory used to resolve relative paths.
    WorkingDir string
    // TblsConfigPath is the path to .tbls.yml / tbls.yml resolved from CLI or defaults.
    TblsConfigPath string
	// SchemaJSONPath is the path to the tbls-generated schema.json file.
	SchemaJSONPath string
	// OutputDir is the directory where SnapSQL YAML files will be written.
	OutputDir string
	// Include patterns to apply after loading tbls JSON. Patterns use the same glob semantics as the legacy pull command.
	Include []string
	// Exclude patterns to apply after loading tbls JSON.
	Exclude []string
	// IncludeViews overrides the default behaviour of importing views when non-nil.
	IncludeViews *bool
	// IncludeIndexes overrides the default behaviour of importing indexes when non-nil.
	IncludeIndexes *bool
	// SchemaAware controls whether YAML files are emitted under schema-aware directories when non-nil.
	SchemaAware *bool
    // DryRun indicates the command should report resolved inputs without writing files.
    DryRun bool
    // Experimental gates the command behind an opt-in flag during rollout.
    Experimental bool
    // Verbose toggles detailed logging.
    Verbose bool
    // Logger, when non-nil, is used for verbose logging.
    Logger func(format string, args ...any)
}
