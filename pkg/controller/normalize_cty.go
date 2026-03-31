// SPDX-FileCopyrightText: 2023 The Crossplane Authors <https://crossplane.io>
//
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// normalizeNullValues recursively normalizes null cty.Values to empty objects
// for nested blocks. This is needed because terraform-plugin-sdk resources that
// iterate over nested blocks (using ElementIterator) will panic if given a null value.
// Only NestingSingle/NestingGroup blocks are normalized; Lists/Sets stay null as-is.
func normalizeNullValues(val cty.Value, schemaMap map[string]*schema.Schema) cty.Value {
	if !val.IsKnown() {
		return val
	}

	// If the value is null and we have a schema, try to normalize it
	if val.IsNull() {
		// For object types, return an empty object with the correct type
		if val.Type().IsObjectType() {
			return cty.EmptyObjectVal
		}
		return val
	}

	// Walk through object attributes and normalize null nested blocks
	if val.Type().IsObjectType() {
		newAttrs := make(map[string]cty.Value, len(schemaMap))

		for name := range schemaMap {
			attrVal := val.GetAttr(name)
			sch := schemaMap[name]

			// Only normalize nested blocks (value of type *schema.Resource)
			// that are NestingSingle or NestingGroup (which appear as object types)
			if sch != nil && sch.Elem != nil {
				if _, ok := sch.Elem.(*schema.Resource); ok {
					// For nested blocks, normalize null nested objects
					newAttrs[name] = normalizeNullValueForSchema(attrVal, sch)
					continue
				}
			}

			newAttrs[name] = attrVal
		}

		return cty.ObjectVal(newAttrs)
	}

	return val
}

// normalizeNullValueForSchema normalizes a cty.Value based on its schema.
// Only normalizes nested object blocks (NestingSingle/NestingGroup), not Lists/Sets.
func normalizeNullValueForSchema(val cty.Value, sch *schema.Schema) cty.Value {
	if !val.IsKnown() {
		return val
	}

	// If the value is null and it's a nested block, normalize to empty object
	if val.IsNull() {
		if sch.Elem != nil {
			if _, ok := sch.Elem.(*schema.Resource); ok {
				// This is a nested block - normalize null to empty object
				return cty.EmptyObjectVal
			}
		}
		// Lists/Sets stay null as-is
		return val
	}

	// For list/set/map types, recursively normalize nested objects within
	switch sch.Type {
	case schema.TypeList, schema.TypeSet:
		if sch.Elem == nil {
			return val
		}

		if res, ok := sch.Elem.(*schema.Resource); ok {
			if val.Type().IsListType() || val.Type().IsTupleType() || val.Type().IsSetType() {
				// Don't convert null/empty to empty collection - keep semantics
				if val.LengthInt() == 0 {
					return val
				}
				elems := make([]cty.Value, 0, val.LengthInt())
				for it := val.ElementIterator(); it.Next(); {
					_, elemVal := it.Element()
					elems = append(elems, normalizeNullValues(elemVal, res.Schema))
				}

				if val.Type().IsListType() || val.Type().IsTupleType() {
					return cty.ListVal(elems)
				}
				return cty.SetVal(elems)
			}
		}
	case schema.TypeMap:
		if val.Type().IsMapType() || val.Type().IsObjectType() {
			if val.LengthInt() == 0 {
				return val
			}
			newMap := make(map[string]cty.Value)
			for it := val.ElementIterator(); it.Next(); {
				k, v := it.Element()
				newMap[k.AsString()] = v
			}
			if len(newMap) == 0 {
				return cty.MapValEmpty(cty.DynamicPseudoType)
			}
			return cty.ObjectVal(newMap)
		}
	}

	return val
}
