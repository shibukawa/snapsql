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
	"fmt"
	"reflect"
	"strings"
)

// HierarchicalMapper handles double underscore field mapping for JOIN queries
type HierarchicalMapper[T any] struct {
	currentEntities map[string]*T
	rootKeyFields   []string
}

// NewHierarchicalMapper creates a new hierarchical mapper
func NewHierarchicalMapper[T any](rootKeyFields ...string) *HierarchicalMapper[T] {
	if len(rootKeyFields) == 0 {
		rootKeyFields = []string{"id"} // Default to id field
	}
	return &HierarchicalMapper[T]{
		currentEntities: make(map[string]*T),
		rootKeyFields:   rootKeyFields,
	}
}

// ProcessFlatRow processes a flat database row into hierarchical structure
func (h *HierarchicalMapper[T]) ProcessFlatRow(flatRow any) ([]T, error) {
	// Extract root entity key and hierarchical data
	rootKey, rootEntity, childData, err := h.extractHierarchicalData(flatRow)
	if err != nil {
		return nil, fmt.Errorf("failed to extract hierarchical data: %w", err)
	}

	var completedEntities []T

	// Check if we have a new root entity
	if existingEntity, exists := h.currentEntities[rootKey]; exists {
		// Same root entity, merge child data
		if err := h.mergeChildData(existingEntity, childData); err != nil {
			return nil, fmt.Errorf("failed to merge child data: %w", err)
		}
	} else {
		// New root entity - yield previous entities
		for _, entity := range h.currentEntities {
			completedEntities = append(completedEntities, *entity)
		}

		// Reset and store new entity
		h.currentEntities = make(map[string]*T)
		h.currentEntities[rootKey] = rootEntity

		if err := h.mergeChildData(rootEntity, childData); err != nil {
			return nil, fmt.Errorf("failed to merge initial child data: %w", err)
		}
	}

	return completedEntities, nil
}

// Finalize returns any remaining entities
func (h *HierarchicalMapper[T]) Finalize() []T {
	var entities []T
	for _, entity := range h.currentEntities {
		entities = append(entities, *entity)
	}
	return entities
}

// extractHierarchicalData extracts root entity and child data from flat row
func (h *HierarchicalMapper[T]) extractHierarchicalData(flatRow any) (string, *T, map[string]any, error) {
	rowValue := reflect.ValueOf(flatRow)
	rowType := reflect.TypeOf(flatRow)

	if rowType.Kind() == reflect.Ptr {
		rowValue = rowValue.Elem()
		rowType = rowType.Elem()
	}

	rootFields := make(map[string]any)
	childData := make(map[string]any)

	// Separate root fields from hierarchical fields (containing __)
	for i := 0; i < rowType.NumField(); i++ {
		field := rowType.Field(i)
		dbTag := field.Tag.Get("db")
		if dbTag == "" {
			continue
		}

		fieldValue := rowValue.Field(i).Interface()

		if strings.Contains(dbTag, "__") {
			// Hierarchical field
			childData[dbTag] = fieldValue
		} else {
			// Root field
			rootFields[dbTag] = fieldValue
		}
	}

	// Create root entity
	rootEntity := new(T)
	if err := mapFieldsToStruct(rootEntity, rootFields); err != nil {
		return "", nil, nil, fmt.Errorf("failed to map root fields: %w", err)
	}

	// Generate root key for grouping
	rootKey := h.generateRootKey(rootFields)

	return rootKey, rootEntity, childData, nil
}

// generateRootKey creates a unique key for grouping based on root key fields
func (h *HierarchicalMapper[T]) generateRootKey(rootFields map[string]any) string {
	var keyParts []string
	for _, keyField := range h.rootKeyFields {
		if value, exists := rootFields[keyField]; exists {
			keyParts = append(keyParts, fmt.Sprintf("%s:%v", keyField, value))
		}
	}
	return strings.Join(keyParts, "|")
}

// mergeChildData merges child data into the root entity
func (h *HierarchicalMapper[T]) mergeChildData(rootEntity *T, childData map[string]any) error {
	// Group child data by path (e.g., "orders__id", "orders__total" -> "orders")
	pathGroups := make(map[string]map[string]any)

	for dbField, value := range childData {
		parts := strings.Split(dbField, "__")
		if len(parts) < 2 {
			continue
		}

		path := parts[0]
		fieldName := strings.Join(parts[1:], "__")

		if pathGroups[path] == nil {
			pathGroups[path] = make(map[string]any)
		}
		pathGroups[path][fieldName] = value
	}

	// Merge each path group into the root entity
	for path, fields := range pathGroups {
		if err := h.mergePathGroup(rootEntity, path, fields); err != nil {
			return fmt.Errorf("failed to merge path group %s: %w", path, err)
		}
	}

	return nil
}

// mergePathGroup merges a specific path group into the entity
func (h *HierarchicalMapper[T]) mergePathGroup(entity *T, path string, fields map[string]any) error {
	entityValue := reflect.ValueOf(entity).Elem()
	entityType := entityValue.Type()

	// Find the field corresponding to this path
	var pathField reflect.Value
	var pathFieldType reflect.Type

	for i := 0; i < entityType.NumField(); i++ {
		field := entityType.Field(i)
		jsonTag := field.Tag.Get("json")

		if jsonTag == path || strings.Split(jsonTag, ",")[0] == path {
			pathField = entityValue.Field(i)
			pathFieldType = field.Type
			break
		}
	}

	if !pathField.IsValid() {
		return fmt.Errorf("path field %s not found in struct", path)
	}

	// Handle slice fields (arrays)
	if pathFieldType.Kind() == reflect.Slice {
		return h.mergeSliceField(pathField, pathFieldType, fields)
	}

	// Handle single object fields
	return h.mergeSingleField(pathField, pathFieldType, fields)
}

// mergeSliceField handles slice/array fields
func (h *HierarchicalMapper[T]) mergeSliceField(pathField reflect.Value, pathFieldType reflect.Type, fields map[string]any) error {
	// Check if all field values are nil (no child data)
	hasData := false
	for _, value := range fields {
		if value != nil {
			hasData = true
			break
		}
	}

	if !hasData {
		return nil // No child data to merge
	}

	// Get the element type of the slice
	elemType := pathFieldType.Elem()

	// Create new element
	newElem := reflect.New(elemType).Elem()

	// Map fields to the new element
	if err := mapFieldsToReflectValue(newElem, fields); err != nil {
		return fmt.Errorf("failed to map fields to slice element: %w", err)
	}

	// Check if this element already exists in the slice (based on ID or other unique field)
	existingSlice := pathField
	for i := 0; i < existingSlice.Len(); i++ {
		existingElem := existingSlice.Index(i)
		if h.elementsEqual(existingElem, newElem) {
			// Update existing element
			existingSlice.Index(i).Set(newElem)
			return nil
		}
	}

	// Add new element to slice
	newSlice := reflect.Append(existingSlice, newElem)
	pathField.Set(newSlice)

	return nil
}

// mergeSingleField handles single object fields
func (h *HierarchicalMapper[T]) mergeSingleField(pathField reflect.Value, pathFieldType reflect.Type, fields map[string]any) error {
	// Check if all field values are nil (no child data)
	hasData := false
	for _, value := range fields {
		if value != nil {
			hasData = true
			break
		}
	}

	if !hasData {
		return nil // No child data to merge
	}

	// Create new object
	newObj := reflect.New(pathFieldType).Elem()

	// Map fields to the new object
	if err := mapFieldsToReflectValue(newObj, fields); err != nil {
		return fmt.Errorf("failed to map fields to single object: %w", err)
	}

	pathField.Set(newObj)
	return nil
}

// elementsEqual checks if two elements are equal based on ID field
func (h *HierarchicalMapper[T]) elementsEqual(elem1, elem2 reflect.Value) bool {
	// Try to find ID field
	for i := 0; i < elem1.NumField(); i++ {
		field := elem1.Type().Field(i)
		dbTag := field.Tag.Get("db")
		if dbTag == "id" {
			val1 := elem1.Field(i).Interface()
			val2 := elem2.Field(i).Interface()
			return val1 == val2
		}
	}
	return false
}

// mapFieldsToStruct maps field values to a struct
func mapFieldsToStruct(dest any, fields map[string]any) error {
	destValue := reflect.ValueOf(dest).Elem()
	return mapFieldsToReflectValue(destValue, fields)
}

// mapFieldsToReflectValue maps field values to a reflect.Value
func mapFieldsToReflectValue(destValue reflect.Value, fields map[string]any) error {
	destType := destValue.Type()

	for i := 0; i < destType.NumField(); i++ {
		field := destType.Field(i)
		dbTag := field.Tag.Get("db")
		if dbTag == "" {
			continue
		}

		if value, exists := fields[dbTag]; exists && value != nil {
			fieldValue := destValue.Field(i)
			if fieldValue.CanSet() {
				// Handle type conversion
				convertedValue, err := convertValue(value, fieldValue.Type())
				if err != nil {
					return fmt.Errorf("failed to convert value for field %s: %w", field.Name, err)
				}
				fieldValue.Set(convertedValue)
			}
		}
	}

	return nil
}

// convertValue converts a value to the target type
func convertValue(value any, targetType reflect.Type) (reflect.Value, error) {
	valueType := reflect.TypeOf(value)

	// Handle nil values for pointer types
	if value == nil {
		if targetType.Kind() == reflect.Ptr {
			return reflect.Zero(targetType), nil
		}
		return reflect.Value{}, fmt.Errorf("cannot assign nil to non-pointer type %s", targetType)
	}

	// Direct assignment if types match
	if valueType.AssignableTo(targetType) {
		return reflect.ValueOf(value), nil
	}

	// Handle pointer types
	if targetType.Kind() == reflect.Ptr {
		elemType := targetType.Elem()
		if valueType.AssignableTo(elemType) {
			ptrValue := reflect.New(elemType)
			ptrValue.Elem().Set(reflect.ValueOf(value))
			return ptrValue, nil
		}
	}

	// Handle conversion from pointer to value
	if valueType.Kind() == reflect.Ptr && !reflect.ValueOf(value).IsNil() {
		elemValue := reflect.ValueOf(value).Elem()
		if elemValue.Type().AssignableTo(targetType) {
			return elemValue, nil
		}
	}

	return reflect.Value{}, fmt.Errorf("cannot convert %s to %s", valueType, targetType)
}
