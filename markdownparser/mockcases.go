package markdownparser

import (
	"time"

	"github.com/shibukawa/snapsql"
)

// ExtractMockTestCases builds mock test cases from a parsed SnapSQL document.
func ExtractMockTestCases(doc *SnapSQLDocument) []snapsql.MockTestCase {
	if doc == nil || len(doc.TestCases) == 0 {
		return nil
	}

	cases := make([]snapsql.MockTestCase, 0, len(doc.TestCases))

	for _, tc := range doc.TestCases {
		response := snapsql.MockResponse{}
		response.Expected = cloneSliceMap(tc.ExpectedResult)

		if len(tc.ExpectedResults) > 0 {
			tables := make([]snapsql.MockTableExpectation, 0, len(tc.ExpectedResults))
			for _, spec := range tc.ExpectedResults {
				tables = append(tables, snapsql.MockTableExpectation{
					Table:        spec.TableName,
					Strategy:     spec.Strategy,
					Rows:         cloneSliceMap(spec.Data),
					ExternalFile: spec.ExternalFile,
				})
			}

			response.Tables = tables
		}

		if tc.ExpectedError != nil {
			message := *tc.ExpectedError
			response.Error = &snapsql.MockError{Message: message}
		}

		responses := make([]snapsql.MockResponse, 0, 1)
		if len(response.Expected) > 0 || len(response.Tables) > 0 || response.Error != nil {
			responses = append(responses, response)
		}

		mockCase := snapsql.MockTestCase{
			Name:          tc.Name,
			Parameters:    cloneMapAny(tc.Parameters),
			Responses:     responses,
			ResultOrdered: tc.ResultOrdered,
			VerifyQuery:   tc.VerifyQuery,
		}

		if tc.SlowQueryThreshold > 0 {
			mockCase.SlowQueryMilli = int64(tc.SlowQueryThreshold / time.Millisecond)
		}

		if mockCase.Responses == nil {
			mockCase.Responses = make([]snapsql.MockResponse, 0)
		}

		cases = append(cases, mockCase)
	}

	return cases
}

func cloneMapAny(src map[string]any) map[string]any {
	if len(src) == 0 {
		return nil
	}

	dup := make(map[string]any, len(src))
	for k, v := range src {
		dup[k] = v
	}

	return dup
}

func cloneSliceMap(src []map[string]any) []map[string]any {
	if len(src) == 0 {
		return nil
	}

	dup := make([]map[string]any, len(src))
	for i, row := range src {
		if row == nil {
			dup[i] = nil
			continue
		}

		copied := make(map[string]any, len(row))
		for k, v := range row {
			copied[k] = v
		}

		dup[i] = copied
	}

	return dup
}
