// SPDX-FileCopyrightText: 2024 The Crossplane Authors <https://crossplane.io>
//
// SPDX-License-Identifier: Apache-2.0

// Package terraform contains Terraform-specific utilities.
package terraform

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// NormalizeParams normalizes a parameters map by converting null/missing
// values for optional TypeList/TypeSet blocks to empty slices.
// This ensures compatibility with Terraform providers that expect empty lists
// rather than null values for optional nested blocks.
//
// This should be called before converting the params map to cty.Value to prevent
// panics when providers call ForEachElement on null collections.
func NormalizeParams(params map[string]any, resourceSchema *schema.Resource) map[string]any {
	if params == nil {
		return params
	}
	if resourceSchema == nil {
		return params
	}
	return normalizeParamsMap(params, resourceSchema.Schema)
}

// normalizeParamsMap recursively normalizes a parameters map based on the schema.
func normalizeParamsMap(params map[string]any, schemaMap map[string]*schema.Schema) map[string]any {
	if params == nil {
		return params
	}

	result := make(map[string]any, len(params))

	for key, paramVal := range params {
		result[key] = paramVal
	}

	// Check for missing optional TypeList/TypeSet fields and add empty slices
	for key, sch := range schemaMap {
		if _, exists := result[key]; !exists {
			if isOptionalListOrSet(sch) {
				result[key] = []any{}
			}
		} else if isOptionalListOrSet(sch) {
			// Normalize existing list/set values recursively
			result[key] = normalizeListOrSetValue(result[key], sch)
		}
	}

	return result
}

// normalizeListOrSetValue normalizes a TypeList or TypeSet value.
func normalizeListOrSetValue(val any, sch *schema.Schema) any {
	if val == nil {
		return []any{}
	}

	listVal, ok := val.([]any)
	if !ok {
		return val
	}

	if len(listVal) == 0 {
		return listVal
	}

	// Check if this is a list of resources (nested blocks)
	if res, ok := sch.Elem.(*schema.Resource); ok {
		normalizedList := make([]any, len(listVal))
		for i, item := range listVal {
			if itemMap, ok := item.(map[string]any); ok {
				normalizedList[i] = normalizeParamsMap(itemMap, res.Schema)
			} else {
				// Handle the case where item is not a map but should be normalized
				// by checking if the item itself is nil and needs to be a map
				normalizedList[i] = item
			}
		}
		return normalizedList
	}

	// For primitive element types, return as-is
	return listVal
}

// isOptionalListOrSet checks if a schema is an optional list or set.
func isOptionalListOrSet(sch *schema.Schema) bool {
	if sch == nil {
		return false
	}
	if !sch.Optional {
		return false
	}
	return sch.Type == schema.TypeList || sch.Type == schema.TypeSet
}
