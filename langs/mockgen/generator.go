package mockgen

import (
	"encoding/json"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/intermediate"
)

const DefaultOutputDir = "testdata/mock"

// Generator writes mock JSON files from intermediate artifacts.
type Generator struct {
	OutputDir         string
	PreserveHierarchy bool
	IntermediateRoot  string
	GenerateEmbed     bool
	EmbedPackage      string
	EmbedFile         string
	generatedPaths    []string
}

// Generate reads the given intermediate JSON file and writes the corresponding mock JSON.
func (g *Generator) Generate(intermediatePath string) (string, error) {
	data, err := os.ReadFile(intermediatePath)
	if err != nil {
		return "", fmt.Errorf("mockgen: failed to read intermediate file %s: %w", intermediatePath, err)
	}

	var format intermediate.IntermediateFormat
	if err := json.Unmarshal(data, &format); err != nil {
		return "", fmt.Errorf("mockgen: failed to unmarshal intermediate file %s: %w", intermediatePath, err)
	}

	outputDir := g.OutputDir
	if outputDir == "" {
		outputDir = DefaultOutputDir
	}

	baseName := strings.TrimSuffix(filepath.Base(intermediatePath), filepath.Ext(intermediatePath))
	if format.FunctionName != "" {
		baseName = format.FunctionName
	}

	relativeDir := ""

	if g.PreserveHierarchy && g.IntermediateRoot != "" {
		if rel, err := filepath.Rel(g.IntermediateRoot, filepath.Dir(intermediatePath)); err == nil && rel != "." {
			relativeDir = rel
		}
	}

	relativePath := filepath.Join(relativeDir, baseName+".json")
	outputPath := filepath.Join(outputDir, relativePath)

	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return "", fmt.Errorf("mockgen: failed to create directory %s: %w", filepath.Dir(outputPath), err)
	}

	cases := normalizeMockCases(format.MockTestCases)

	encoded, err := json.MarshalIndent(cases, "", "  ")
	if err != nil {
		return "", fmt.Errorf("mockgen: failed to marshal mock cases for %s: %w", intermediatePath, err)
	}

	encoded = append(encoded, '\n')

	if err := os.WriteFile(outputPath, encoded, 0o644); err != nil {
		return "", fmt.Errorf("mockgen: failed to write %s: %w", outputPath, err)
	}

	g.generatedPaths = append(g.generatedPaths, filepath.ToSlash(relativePath))

	return outputPath, nil
}

// Finalize generates the go:embed wrapper when requested.
func (g *Generator) Finalize() (string, error) {
	if !g.GenerateEmbed {
		return "", nil
	}

	if len(g.generatedPaths) == 0 {
		return "", nil
	}

	packageName := g.EmbedPackage
	if packageName == "" {
		packageName = "mock"
	}

	embedFile := g.EmbedFile
	if embedFile == "" {
		embedFile = "mock.go"
	}

	paths := make([]string, 0, len(g.generatedPaths))

	seen := make(map[string]struct{}, len(g.generatedPaths))
	for _, p := range g.generatedPaths {
		if _, ok := seen[p]; ok {
			continue
		}

		seen[p] = struct{}{}
		paths = append(paths, p)
	}

	sort.Strings(paths)

	embedPattern := strings.Join(paths, " ")
	source := fmt.Sprintf(`package %s

import (
	"embed"
	"sync"

	"github.com/shibukawa/snapsql/langs/snapsqlgo"
)

//go:embed %s
var embeddedFiles embed.FS

var (
	providerOnce sync.Once
	provider     snapsqlgo.MockProvider
)

// Provider exposes the embedded mock data as a snapsqlgo.MockProvider.
func Provider() snapsqlgo.MockProvider {
	providerOnce.Do(func() {
		provider = snapsqlgo.NewEmbeddedMockProvider(embeddedFiles)
	})
	return provider
}
`, packageName, embedPattern)

	formatted, err := format.Source([]byte(source))
	if err != nil {
		return "", fmt.Errorf("mockgen: failed to format embed source: %w", err)
	}

	outputPath := filepath.Join(g.OutputDir, embedFile)
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return "", fmt.Errorf("mockgen: failed to create embed directory %s: %w", filepath.Dir(outputPath), err)
	}

	if err := os.WriteFile(outputPath, formatted, 0o644); err != nil {
		return "", fmt.Errorf("mockgen: failed to write embed file %s: %w", outputPath, err)
	}

	return outputPath, nil
}

func normalizeMockCases(source []snapsql.MockTestCase) []snapsql.MockTestCase {
	if len(source) == 0 {
		return make([]snapsql.MockTestCase, 0)
	}

	normalized := make([]snapsql.MockTestCase, len(source))
	for i, c := range source {
		copyCase := c
		if copyCase.Parameters == nil {
			copyCase.Parameters = make(map[string]any)
		}

		if copyCase.Responses == nil {
			copyCase.Responses = make([]snapsql.MockResponse, 0)
		} else {
			responses := make([]snapsql.MockResponse, len(copyCase.Responses))
			for j, resp := range copyCase.Responses {
				copyResp := resp
				if copyResp.Expected == nil {
					copyResp.Expected = make([]map[string]any, 0)
				}

				if copyResp.Tables == nil {
					copyResp.Tables = make([]snapsql.MockTableExpectation, 0)
				}

				responses[j] = copyResp
			}

			copyCase.Responses = responses
		}

		normalized[i] = copyCase
	}

	return normalized
}
