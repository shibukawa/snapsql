package snapsqlgo

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unicode"

	snapsql "github.com/shibukawa/snapsql"
)

// MockCase is an alias of snapsql.MockTestCase for runtime usage.
type MockCase = snapsql.MockTestCase

// MockResponse is an alias of snapsql.MockResponse for runtime usage.
type MockResponse = snapsql.MockResponse

// MockTableExpectation is an alias of snapsql.MockTableExpectation.
type MockTableExpectation = snapsql.MockTableExpectation

// MockSQLResult is an alias of snapsql.MockSQLResult.
type MockSQLResult = snapsql.MockSQLResult

// MockError is an alias of snapsql.MockError.
type MockError = snapsql.MockError

// ErrMock acts as the sentinel for all mock-related errors.
var ErrMock = errors.New("snapsqlgo: mock error")

// ErrMockSequenceDepleted indicates that configured mock scenarios were fully consumed.
var ErrMockSequenceDepleted = errors.New("snapsqlgo: mock sequence depleted")

// ErrMockCaseNotFound indicates that the requested mock case was not found.
var ErrMockCaseNotFound = errors.New("snapsqlgo: mock case not found")

// ErrMockDataNotFound indicates that mock data files could not be located.
var ErrMockDataNotFound = errors.New("snapsqlgo: mock data not found")

// MockOpt customises mock behaviour for a specific invocation sequence.
type MockOpt struct {
	Name         string
	Index        int
	Err          error
	LastInsertID int64
	RowsAffected int64
	NoRepeat     bool
}

// MockExecution represents one mock response consumption.
type MockExecution struct {
	Case     MockCase
	Response *MockResponse
	Err      error
	Opt      MockOpt
}

// SQLResult builds an sql.Result representation if available.
func (m *MockExecution) SQLResult() *MockResult {
	if m == nil {
		return nil
	}

	var (
		rowsAffected int64
		lastInsertID int64
		hasResult    bool
	)

	if m.Response != nil && m.Response.Result != nil {
		if m.Response.Result.RowsAffected != nil {
			rowsAffected = *m.Response.Result.RowsAffected
			hasResult = true
		}

		if m.Response.Result.LastInsertID != nil {
			lastInsertID = *m.Response.Result.LastInsertID
			hasResult = true
		}
	}

	if m.Opt.RowsAffected != 0 || m.Opt.LastInsertID != 0 {
		rowsAffected = m.Opt.RowsAffected
		lastInsertID = m.Opt.LastInsertID
		hasResult = true
	}

	if !hasResult {
		return nil
	}

	return NewMockResult(rowsAffected, lastInsertID)
}

// ExpectedRows returns copies of expected row maps for mapping to structs.
func (m *MockExecution) ExpectedRows() []map[string]any {
	if m == nil || m.Response == nil || len(m.Response.Expected) == 0 {
		return nil
	}

	rows := make([]map[string]any, len(m.Response.Expected))
	for i, row := range m.Response.Expected {
		if row == nil {
			rows[i] = nil
			continue
		}

		normalized := normalizeMockValue(row)
		mapRow, _ := normalized.(map[string]any)
		rows[i] = mapRow
	}

	return rows
}

// MapMockExecutionToStruct converts the mock execution to a typed struct.
func MapMockExecutionToStruct[T any](exec *MockExecution) (T, error) {
	var zero T
	if exec == nil {
		return zero, fmt.Errorf("%w: mock execution is nil", ErrMock)
	}

	rows := exec.ExpectedRows()
	if len(rows) == 0 {
		return zero, fmt.Errorf("%w: mock case %s has no expected rows", ErrMock, exec.Case.Name)
	}

	return MapMockDataToStruct[T](rows[0])
}

// MapMockExecutionToSlice converts the mock execution to a typed slice.
func MapMockExecutionToSlice[T any](exec *MockExecution) ([]T, error) {
	if exec == nil {
		return nil, fmt.Errorf("%w: mock execution is nil", ErrMock)
	}

	rows := exec.ExpectedRows()
	if len(rows) == 0 {
		return nil, fmt.Errorf("%w: mock case %s has no expected rows", ErrMock, exec.Case.Name)
	}

	items := make([]any, len(rows))
	for i, row := range rows {
		items[i] = row
	}

	return MapMockDataToSlice[T](items)
}

type mockScenario struct {
	caseDef       MockCase
	opt           MockOpt
	responseIndex int
}

type mockQueue struct {
	scenarios []*mockScenario
	depleted  bool
}

type mockRegistry struct {
	mu        sync.Mutex
	functions map[string]*mockQueue
}

func newMockRegistry() *mockRegistry {
	return &mockRegistry{functions: make(map[string]*mockQueue)}
}

func (r *mockRegistry) register(functionName string, cases []MockCase, opts ...MockOpt) error {
	if len(cases) == 0 {
		return fmt.Errorf("%w: no mock cases provided for %s", ErrMock, functionName)
	}

	normalized := make([]MockCase, len(cases))
	copy(normalized, cases)

	scenarios := make([]*mockScenario, 0, len(normalized))

	selectCase := func(opt MockOpt) (MockCase, error) {
		if opt.Name != "" {
			for _, c := range normalized {
				if strings.EqualFold(c.Name, opt.Name) {
					return c, nil
				}
			}

			return MockCase{}, fmt.Errorf("%w: %s", ErrMockCaseNotFound, opt.Name)
		}

		idx := opt.Index
		if idx < 0 {
			idx = 0
		}

		if idx >= len(normalized) {
			return MockCase{}, fmt.Errorf("%w: mock case index %d out of range for %s", ErrMock, idx, functionName)
		}

		return normalized[idx], nil
	}

	if len(opts) == 0 {
		for _, c := range normalized {
			scenarios = append(scenarios, &mockScenario{caseDef: c})
		}
	} else {
		for _, opt := range opts {
			c, err := selectCase(opt)
			if err != nil {
				return err
			}

			scenarios = append(scenarios, &mockScenario{caseDef: c, opt: opt})
		}
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	queue, ok := r.functions[functionName]
	if !ok {
		queue = &mockQueue{}
		r.functions[functionName] = queue
	}

	queue.scenarios = append(queue.scenarios, scenarios...)

	return nil
}

func (r *mockRegistry) consume(functionName string) (*MockExecution, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	queue, ok := r.functions[functionName]
	if !ok {
		return nil, false, nil
	}

	if len(queue.scenarios) == 0 {
		if queue.depleted {
			return nil, true, ErrMockSequenceDepleted
		}

		return nil, false, nil
	}

	scenario := queue.scenarios[0]

	exec := &MockExecution{
		Case: scenario.caseDef,
		Opt:  scenario.opt,
	}

	if scenario.opt.Err != nil {
		exec.Err = scenario.opt.Err
		if scenario.opt.NoRepeat {
			queue.scenarios = queue.scenarios[1:]
			if len(queue.scenarios) == 0 {
				queue.depleted = true
			}
		}

		return exec, true, nil
	}

	if len(scenario.caseDef.Responses) == 0 {
		if scenario.opt.NoRepeat {
			queue.scenarios = queue.scenarios[1:]
			if len(queue.scenarios) == 0 {
				queue.depleted = true
			}
		}

		return nil, true, fmt.Errorf("%w: mock case %s has no responses", ErrMock, scenario.caseDef.Name)
	}

	if scenario.responseIndex >= len(scenario.caseDef.Responses) {
		if scenario.opt.NoRepeat {
			queue.scenarios = queue.scenarios[1:]
			if len(queue.scenarios) == 0 {
				queue.depleted = true
			}

			return nil, true, ErrMockSequenceDepleted
		}

		scenario.responseIndex = len(scenario.caseDef.Responses) - 1
	}

	resp := scenario.caseDef.Responses[scenario.responseIndex]
	exec.Response = &resp
	scenario.responseIndex++

	if scenario.responseIndex >= len(scenario.caseDef.Responses) {
		if scenario.opt.NoRepeat {
			queue.scenarios = queue.scenarios[1:]
			if len(queue.scenarios) == 0 {
				queue.depleted = true
			}
		} else {
			scenario.responseIndex = len(scenario.caseDef.Responses) - 1
		}
	}

	return exec, true, nil
}

// WithMock registers mock cases for the specified function.
func WithMock(ctx context.Context, functionName string, cases []MockCase, opts ...MockOpt) (context.Context, error) {
	ctx, ec := withExecutionContext(ctx)

	if ec.mocks == nil {
		ec.mocks = newMockRegistry()
	}

	if err := ec.mocks.register(functionName, cases, opts...); err != nil {
		return ctx, err
	}

	return ctx, nil
}

// MatchMock retrieves the next mock execution for the function, if configured.
func MatchMock(ctx context.Context, functionName string) (*MockExecution, bool, error) {
	ec := ExtractExecutionContext(ctx)
	if ec == nil || ec.mocks == nil {
		return nil, false, nil
	}

	return ec.mocks.consume(functionName)
}

// MockProvider loads mock cases from various sources.
type MockProvider interface {
	Cases(functionName string) ([]MockCase, error)
}

type embeddedMockProvider struct {
	fs    embed.FS
	once  sync.Once
	cache map[string][]MockCase
	err   error
}

// NewEmbeddedMockProvider creates a provider backed by embedded JSON files.
func NewEmbeddedMockProvider(fs embed.FS) MockProvider {
	return &embeddedMockProvider{fs: fs}
}

func (p *embeddedMockProvider) loadAll() {
	p.cache = make(map[string][]MockCase)

	entries, err := fs.Glob(p.fs, "*.json")
	if err != nil {
		p.err = fmt.Errorf("snapsqlgo: failed to enumerate embedded mock files: %w", err)
		return
	}

	for _, entry := range entries {
		data, readErr := p.fs.ReadFile(entry)
		if readErr != nil {
			p.err = fmt.Errorf("snapsqlgo: failed to read embedded mock %s: %w", entry, readErr)
			return
		}

		var cases []MockCase
		if unmarshalErr := json.Unmarshal(data, &cases); unmarshalErr != nil {
			p.err = fmt.Errorf("snapsqlgo: failed to unmarshal embedded mock %s: %w", entry, unmarshalErr)
			return
		}

		key := strings.TrimSuffix(filepath.Base(entry), filepath.Ext(entry))
		for _, alias := range canonicalMockKeys(key) {
			p.cache[alias] = cases
		}
	}
}

func (p *embeddedMockProvider) Cases(functionName string) ([]MockCase, error) {
	p.once.Do(p.loadAll)

	if p.err != nil {
		return nil, p.err
	}

	for _, key := range canonicalMockKeys(functionName) {
		if cases, ok := p.cache[key]; ok {
			return copyCases(cases), nil
		}
	}

	return nil, fmt.Errorf("%w: %s", ErrMockCaseNotFound, functionName)
}

type filesystemMockProvider struct {
	root  string
	cache map[string][]MockCase
	mu    sync.Mutex
}

// NewFilesystemMockProvider locates testdata/mock by walking up to module root.
func NewFilesystemMockProvider(startDir string) (MockProvider, error) {
	dir := startDir
	if dir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("snapsqlgo: failed to obtain working directory: %w", err)
		}

		dir = cwd
	}

	dir, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("snapsqlgo: failed to resolve absolute path: %w", err)
	}

	var last string

	for {
		candidate := filepath.Join(dir, "testdata", "mock")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return &filesystemMockProvider{
				root:  candidate,
				cache: make(map[string][]MockCase),
			}, nil
		}

		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			break
		}

		parent := filepath.Dir(dir)
		if parent == dir || parent == last {
			break
		}

		last = dir
		dir = parent
	}

	return nil, ErrMockDataNotFound
}

func (p *filesystemMockProvider) Cases(functionName string) ([]MockCase, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, key := range canonicalMockKeys(functionName) {
		if cases, ok := p.cache[key]; ok {
			return copyCases(cases), nil
		}
	}

	var (
		path      string
		cacheKeys []string
	)

	for _, key := range canonicalMockKeys(functionName) {
		candidate := filepath.Join(p.root, key+".json")
		if _, err := os.Stat(candidate); err == nil {
			path = candidate
			cacheKeys = canonicalMockKeys(key)

			break
		}
	}

	if path == "" {
		return nil, fmt.Errorf("%w: %s", ErrMockCaseNotFound, functionName)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("snapsqlgo: failed to read mock file %s: %w", path, err)
	}

	var cases []MockCase
	if err := json.Unmarshal(data, &cases); err != nil {
		return nil, fmt.Errorf("snapsqlgo: failed to unmarshal mock file %s: %w", path, err)
	}

	for _, key := range cacheKeys {
		p.cache[key] = cases
	}

	return copyCases(cases), nil
}

// WithMockProvider loads cases via provider then registers them.
func WithMockProvider(ctx context.Context, functionName string, provider MockProvider, opts ...MockOpt) (context.Context, error) {
	cases, err := provider.Cases(functionName)
	if err != nil {
		return ctx, err
	}

	return WithMock(ctx, functionName, cases, opts...)
}

func canonicalMockKeys(name string) []string {
	set := make(map[string]struct{})
	add := func(key string) {
		if key == "" {
			return
		}

		set[key] = struct{}{}
	}

	base := normalizeSeparators(name)
	add(base)
	add(strings.ToLower(base))

	if strings.Contains(base, "_") {
		camel := snakeToUpperCamel(base)
		add(camel)
		add(lowerFirst(camel))
	} else {
		snake := camelToSnake(base)
		add(snake)
		camel := snakeToUpperCamel(snake)
		add(camel)
		add(lowerFirst(camel))
	}

	keys := make([]string, 0, len(set))
	for k := range set {
		keys = append(keys, k)
	}

	return keys
}

func snakeToUpperCamel(s string) string {
	parts := strings.Split(s, "_")

	var b strings.Builder

	for _, part := range parts {
		if part == "" {
			continue
		}

		runes := []rune(strings.ToLower(part))
		runes[0] = unicode.ToUpper(runes[0])
		b.WriteString(string(runes))
	}

	return b.String()
}

func lowerFirst(s string) string {
	if s == "" {
		return s
	}

	return strings.ToLower(s[:1]) + s[1:]
}

func camelToSnake(s string) string {
	if s == "" {
		return s
	}

	var b strings.Builder

	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 {
				b.WriteByte('_')
			}

			b.WriteRune(unicode.ToLower(r))

			continue
		}

		b.WriteRune(r)
	}

	return b.String()
}

func normalizeSeparators(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "-", "_")
	s = strings.ReplaceAll(s, " ", "_")

	s = strings.ReplaceAll(s, "\t", "_")
	for strings.Contains(s, "__") {
		s = strings.ReplaceAll(s, "__", "_")
	}

	return strings.Trim(s, "_")
}

const placeholderTimeValue = "1970-01-01T00:00:00Z"

func normalizeMockValue(value any) any {
	switch v := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(v))
		for k, val := range v {
			out[k] = normalizeMockValue(val)
		}

		return out
	case []any:
		if len(v) == 1 {
			if str, ok := v[0].(string); ok {
				norm := strings.TrimSpace(strings.Trim(str, "[]"))
				switch strings.ToLower(norm) {
				case "currentdate", "now", "today":
					return placeholderTimeValue
				}
			}
		}

		out := make([]any, len(v))
		for i, elem := range v {
			out[i] = normalizeMockValue(elem)
		}

		return out
	case string:
		norm := strings.TrimSpace(strings.Trim(v, "[]"))
		switch strings.ToLower(norm) {
		case "currentdate", "now", "today":
			return placeholderTimeValue
		}

		return v
	default:
		return v
	}
}

func copyCases(source []MockCase) []MockCase {
	if len(source) == 0 {
		return nil
	}

	copied := make([]MockCase, len(source))
	copy(copied, source)

	return copied
}
