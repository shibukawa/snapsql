package pygen

import (
	"errors"
	"testing"

	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/intermediate"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.PackageName != "generated" {
		t.Errorf("DefaultConfig().PackageName = %v, want %v", config.PackageName, "generated")
	}

	if config.OutputPath != "./generated" {
		t.Errorf("DefaultConfig().OutputPath = %v, want %v", config.OutputPath, "./generated")
	}

	if config.Dialect != "postgres" {
		t.Errorf("DefaultConfig().Dialect = %v, want %v", config.Dialect, "postgres")
	}

	if !config.Features.EnableQueryLogging {
		t.Error("DefaultConfig().Features.EnableQueryLogging should be true")
	}

	if !config.Features.PreserveHierarchy {
		t.Error("DefaultConfig().Features.PreserveHierarchy should be true")
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
		errType error
	}{
		{
			name: "valid config",
			config: Config{
				PackageName: "mypackage",
				OutputPath:  "./output",
				Dialect:     snapsql.DialectPostgres,
			},
			wantErr: false,
		},
		{
			name: "missing dialect",
			config: Config{
				PackageName: "mypackage",
				OutputPath:  "./output",
				Dialect:     "",
			},
			wantErr: true,
			errType: ErrDialectRequired,
		},
		{
			name: "unsupported dialect",
			config: Config{
				PackageName: "mypackage",
				OutputPath:  "./output",
				Dialect:     "oracle",
			},
			wantErr: true,
			errType: ErrUnsupportedDialect,
		},
		{
			name: "empty package name gets default",
			config: Config{
				PackageName: "",
				OutputPath:  "./output",
				Dialect:     snapsql.DialectMySQL,
			},
			wantErr: false,
		},
		{
			name: "empty output path gets default",
			config: Config{
				PackageName: "mypackage",
				OutputPath:  "",
				Dialect:     snapsql.DialectSQLite,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && !errors.Is(err, tt.errType) {
				t.Errorf("Config.Validate() error = %v, want %v", err, tt.errType)
			}

			// Check defaults are applied
			if !tt.wantErr {
				if tt.config.PackageName == "" {
					t.Error("Config.Validate() should set default PackageName")
				}

				if tt.config.OutputPath == "" {
					t.Error("Config.Validate() should set default OutputPath")
				}
			}
		})
	}
}

func TestNewGeneratorFromConfig(t *testing.T) {
	format := &intermediate.IntermediateFormat{
		FunctionName: "test_func",
	}

	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: Config{
				PackageName: "queries",
				OutputPath:  "./output",
				Dialect:     snapsql.DialectPostgres,
				MockPath:    "./mocks",
			},
			wantErr: false,
		},
		{
			name: "invalid config - missing dialect",
			config: Config{
				PackageName: "queries",
				OutputPath:  "./output",
				Dialect:     "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewGeneratorFromConfig(format, tt.config)

			if (err != nil) != tt.wantErr {
				t.Errorf("NewGeneratorFromConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if got.PackageName != tt.config.PackageName {
					t.Errorf("Generator.PackageName = %v, want %v", got.PackageName, tt.config.PackageName)
				}

				if got.Dialect != tt.config.Dialect {
					t.Errorf("Generator.Dialect = %v, want %v", got.Dialect, tt.config.Dialect)
				}

				if got.OutputPath != tt.config.OutputPath {
					t.Errorf("Generator.OutputPath = %v, want %v", got.OutputPath, tt.config.OutputPath)
				}

				if got.MockPath != tt.config.MockPath {
					t.Errorf("Generator.MockPath = %v, want %v", got.MockPath, tt.config.MockPath)
				}
			}
		})
	}
}

func TestConfig_ApplyToGenerator(t *testing.T) {
	format := &intermediate.IntermediateFormat{
		FunctionName: "test_func",
	}

	gen := New(format)

	config := Config{
		PackageName: "custom",
		OutputPath:  "./custom_output",
		Dialect:     snapsql.DialectMySQL,
		MockPath:    "./custom_mocks",
	}

	config.ApplyToGenerator(gen)

	if gen.PackageName != config.PackageName {
		t.Errorf("ApplyToGenerator() PackageName = %v, want %v", gen.PackageName, config.PackageName)
	}

	if gen.OutputPath != config.OutputPath {
		t.Errorf("ApplyToGenerator() OutputPath = %v, want %v", gen.OutputPath, config.OutputPath)
	}

	if gen.Dialect != config.Dialect {
		t.Errorf("ApplyToGenerator() Dialect = %v, want %v", gen.Dialect, config.Dialect)
	}

	if gen.MockPath != config.MockPath {
		t.Errorf("ApplyToGenerator() MockPath = %v, want %v", gen.MockPath, config.MockPath)
	}
}
