// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package snapsqlgo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAsIterableAny(t *testing.T) {
	t.Run("string slice", func(t *testing.T) {
		input := []string{"user1", "user2", "user3"}

		var result []any
		for item := range AsIterableAny(input) {
			result = append(result, item)
		}

		assert.Equal(t, 3, len(result))
		assert.Equal(t, "user1", result[0])
		assert.Equal(t, "user2", result[1])
		assert.Equal(t, "user3", result[2])
	})

	t.Run("int slice", func(t *testing.T) {
		input := []int{1, 2, 3, 4, 5}

		var result []any
		for item := range AsIterableAny(input) {
			result = append(result, item)
		}

		assert.Equal(t, 5, len(result))
		assert.Equal(t, 1, result[0])
		assert.Equal(t, 2, result[1])
		assert.Equal(t, 5, result[4])
	})

	t.Run("any slice", func(t *testing.T) {
		input := []any{"mixed", 42, true}

		var result []any
		for item := range AsIterableAny(input) {
			result = append(result, item)
		}

		assert.Equal(t, 3, len(result))
		assert.Equal(t, "mixed", result[0])
		assert.Equal(t, 42, result[1])
		assert.Equal(t, true, result[2])
	})

	t.Run("empty slice", func(t *testing.T) {
		input := []string{}

		var result []any
		for item := range AsIterableAny(input) {
			result = append(result, item)
		}

		assert.Equal(t, 0, len(result))
	})

	t.Run("nil value", func(t *testing.T) {
		var result []any
		for item := range AsIterableAny(nil) {
			result = append(result, item)
		}

		assert.Equal(t, 0, len(result))
	})

	t.Run("non-slice value", func(t *testing.T) {
		input := "not a slice"

		var result []any
		for item := range AsIterableAny(input) {
			result = append(result, item)
		}

		// Non-slice values are yielded as-is
		assert.Equal(t, 1, len(result))
		assert.Equal(t, "not a slice", result[0])
	})

	t.Run("early break", func(t *testing.T) {
		input := []string{"a", "b", "c", "d", "e"}

		var result []any
		for item := range AsIterableAny(input) {
			result = append(result, item)
			if len(result) >= 3 {
				break
			}
		}

		assert.Equal(t, 3, len(result))
		assert.Equal(t, "a", result[0])
		assert.Equal(t, "b", result[1])
		assert.Equal(t, "c", result[2])
	})
}
