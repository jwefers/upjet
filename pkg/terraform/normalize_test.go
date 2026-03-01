// SPDX-FileCopyrightText: 2024 The Crossplane Authors <https://crossplane.io>
//
// SPDX-License-Identifier: Apache-2.0

package terraform

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/assert"
)

func TestNormalizeParams_NullOptionalBlock(t *testing.T) {
	resourceSchema := &schema.Resource{
		Schema: map[string]*schema.Schema{
			"name":     {Type: schema.TypeString, Required: true},
			"strategy": {Type: schema.TypeString, Required: true},
			"options": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"password_policy": {Type: schema.TypeString, Optional: true},
						"attributes": {
							Type:     schema.TypeList,
							Optional: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"email": {
										Type:     schema.TypeList,
										Optional: true,
										MaxItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"identifier": {
													Type:     schema.TypeList,
													Optional: true,
													MaxItems: 1,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"active": {Type: schema.TypeBool, Optional: true},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	params := map[string]any{
		"name":     "test",
		"strategy": "auth0",
	}

	normalized := NormalizeParams(params, resourceSchema)
	assert.NotNil(t, normalized)
	assert.Equal(t, "test", normalized["name"])
	assert.Equal(t, "auth0", normalized["strategy"])
	assert.Equal(t, []any{}, normalized["options"], "options should be normalized to empty slice")
}

func TestNormalizeParams_PopulatedNestedBlocks(t *testing.T) {
	resourceSchema := &schema.Resource{
		Schema: map[string]*schema.Schema{
			"name": {Type: schema.TypeString, Required: true},
			"options": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"password_policy": {Type: schema.TypeString, Optional: true},
					},
				},
			},
		},
	}

	params := map[string]any{
		"name": "test",
		"options": []any{
			map[string]any{
				"password_policy": "good",
			},
		},
	}

	normalized := NormalizeParams(params, resourceSchema)
	assert.NotNil(t, normalized)
	assert.Equal(t, "test", normalized["name"])

	options, ok := normalized["options"].([]any)
	assert.True(t, ok)
	assert.Equal(t, 1, len(options))

	optionsMap, ok := options[0].(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "good", optionsMap["password_policy"])
}

func TestNormalizeParams_NestedMissingFields(t *testing.T) {
	resourceSchema := &schema.Resource{
		Schema: map[string]*schema.Schema{
			"name": {Type: schema.TypeString, Required: true},
			"options": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"password_policy": {Type: schema.TypeString, Optional: true},
						"attributes": {
							Type:     schema.TypeList,
							Optional: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"email": {
										Type:     schema.TypeList,
										Optional: true,
										MaxItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"identifier": {
													Type:     schema.TypeList,
													Optional: true,
													MaxItems: 1,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"active": {Type: schema.TypeBool, Optional: true},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	params := map[string]any{
		"name": "test",
		"options": []any{
			map[string]any{
				// password_policy is missing - string type, not normalized
				"attributes": []any{
					map[string]any{
						// email is missing - list type, should be normalized
					},
				},
			},
		},
	}

	normalized := NormalizeParams(params, resourceSchema)
	assert.NotNil(t, normalized)

	options := normalized["options"].([]any)
	optionsMap := options[0].(map[string]any)
	// password_policy is a string type, not a list, so it should not be auto-populated
	_, hasPasswordPolicy := optionsMap["password_policy"]
	assert.False(t, hasPasswordPolicy, "password_policy (string type) should not be auto-populated")

	attrs := optionsMap["attributes"].([]any)
	attrsMap := attrs[0].(map[string]any)
	assert.Equal(t, []any{}, attrsMap["email"], "email should be empty slice")
}

func TestNormalizeParams_NilParams(t *testing.T) {
	resourceSchema := &schema.Resource{
		Schema: map[string]*schema.Schema{
			"name": {Type: schema.TypeString, Required: true},
		},
	}

	normalized := NormalizeParams(nil, resourceSchema)
	assert.Nil(t, normalized)
}

func TestNormalizeParams_NilResource(t *testing.T) {
	params := map[string]any{
		"name": "test",
	}

	normalized := NormalizeParams(params, nil)
	assert.Equal(t, params, normalized)
}

func TestNormalizeParams_RequiredListNotNormalized(t *testing.T) {
	resourceSchema := &schema.Resource{
		Schema: map[string]*schema.Schema{
			"name": {Type: schema.TypeString, Required: true},
			"tags": {
				Type:     schema.TypeList,
				Required: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
		},
	}

	params := map[string]any{
		"name": "test",
		// tags is missing but required - should not be added
	}

	normalized := NormalizeParams(params, resourceSchema)
	assert.NotNil(t, normalized)
	_, exists := normalized["tags"]
	assert.False(t, exists, "required fields should not be auto-populated")
}
