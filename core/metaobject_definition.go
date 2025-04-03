package core

import (
	"encoding/json"
	"strings"

	"github.com/JohnnyMcGee/metaobjects-cli/shopify"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type FieldDefinition struct {
	Type        string         `json:"type"`
	Description string         `json:"description,omitempty"`
	Required    bool           `json:"required,omitempty"`
	Validations map[string]any `json:"validations,omitempty"`
}

type OnlineStoreCapabilities struct {
	CanCreateRedirects bool   `json:"canCreateRedirects,omitempty"`
	UrlHandle          string `json:"urlHandle,omitempty"`
}

type RenderableCapabilities struct {
	MetaDescriptionKey string `json:"metaDescriptionKey,omitempty"`
	MetaTitleKey       string `json:"metaTitleKey,omitempty"`
}

type Capabilities struct {
	OnlineStore  *OnlineStoreCapabilities `json:"onlineStore,omitempty"`
	Publishable  bool                     `json:"publishable,omitempty"`
	Renderable   *RenderableCapabilities  `json:"renderable,omitempty"`
	Translatable bool                     `json:"translatable,omitempty"`
}

type Access struct {
	Admin      shopify.MetaobjectAdminAccess      `json:"admin,omitempty"`
	Storefront shopify.MetaobjectStorefrontAccess `json:"storefront,omitempty"`
}

type MetaobjectDefinition struct {
	Name             string                     `json:"name,omitempty"`
	Description      string                     `json:"description,omitempty"`
	Access           *Access                    `json:"access,omitempty"`
	Capabilities     *Capabilities              `json:"capabilities,omitempty"`
	DisplayNameKey   string                     `json:"displayNameKey,omitempty"`
	FieldDefinitions map[string]FieldDefinition `json:"fieldDefinitions"`
}

func convertAccess(access shopify.Cli_MetaobjectDefinitionAccessMetaobjectAccess) (a *Access, empty bool) {
	a = &Access{}
	empty = true

	if access.Admin != shopify.MetaobjectAdminAccessPublicReadWrite {
		a.Admin, empty = access.Admin, false
	}

	if access.Storefront != shopify.MetaobjectStorefrontAccessPublicRead {
		a.Storefront, empty = access.Storefront, false
	}

	return a, empty
}

func convertFieldDefinition(definition shopify.Cli_MetaobjectDefinitionFieldDefinitionsMetaobjectFieldDefinition) FieldDefinition {
	f := FieldDefinition{
		Type:        definition.Type.Name,
		Description: definition.Description,
		Required:    definition.Required,
	}

	if len(definition.Validations) > 0 {
		validations := make(map[string]any, len(definition.Validations))

		for _, v := range definition.Validations {
			b := []byte(v.Value)
			var value any

			if err := json.Unmarshal(b, &value); err != nil {
				value = v.Value
			}

			validations[v.Name] = value
		}

		f.Validations = validations
	}

	return f
}

func convertCapabilities(capabilities shopify.Cli_MetaobjectDefinitionCapabilitiesMetaobjectCapabilities) (cap *Capabilities, empty bool) {
	cap = &Capabilities{}
	empty = true

	if capabilities.OnlineStore.Enabled {
		cap.OnlineStore, empty = &OnlineStoreCapabilities{
			CanCreateRedirects: capabilities.OnlineStore.Data.CanCreateRedirects,
			UrlHandle:          capabilities.OnlineStore.Data.UrlHandle,
		}, false
	}

	if capabilities.Renderable.Enabled {
		cap.Renderable, empty = &RenderableCapabilities{
			MetaDescriptionKey: capabilities.Renderable.Data.MetaDescriptionKey,
			MetaTitleKey:       capabilities.Renderable.Data.MetaTitleKey,
		}, false
	}

	if capabilities.Publishable.Enabled {
		cap.Publishable, empty = true, false
	}

	if capabilities.Translatable.Enabled {
		cap.Translatable, empty = true, false
	}

	return cap, empty
}

func ConvertMetaobjectDefinition(definition shopify.Cli_MetaobjectDefinition) MetaobjectDefinition {
	d := MetaobjectDefinition{
		Name:             definition.Name,
		Description:      definition.Description,
		DisplayNameKey:   definition.DisplayNameKey,
		FieldDefinitions: make(map[string]FieldDefinition, len(definition.FieldDefinitions)),
	}

	c := cases.Title(language.English)
	title := c.String(strings.ReplaceAll(strings.ReplaceAll(definition.Type, "_", " "), "-", " "))

	if definition.Name == title {
		d.Name = ""
	}

	var defaultDisplayNameKey string

	for _, f := range definition.FieldDefinitions {

		d.FieldDefinitions[f.Key] = convertFieldDefinition(f)

		if defaultDisplayNameKey == "" && f.Type.Name == "single_line_text_field" {
			defaultDisplayNameKey = f.Key
		}
	}

	if definition.DisplayNameKey == defaultDisplayNameKey {
		d.DisplayNameKey = ""
	}

	if cap, empty := convertCapabilities(definition.Capabilities); !empty {
		d.Capabilities = cap
	}

	if access, empty := convertAccess(definition.Access); !empty {
		d.Access = access
	}

	return d
}

func CreateMetaobjectDefinitionMap(definitions []shopify.Cli_MetaobjectDefinition) map[string]MetaobjectDefinition {
	definitionMap := make(map[string]MetaobjectDefinition, len(definitions))

	referenceTypes := make(map[string]string)

	for _, definition := range definitions {
		definitionMap[definition.Type] = ConvertMetaobjectDefinition(definition)
		referenceTypes[definition.Id] = definition.Type
	}

	// Normalize metaobject definition references to use the type name
	// instead of the ID. This allows us to use the same definitions
	// across different stores and environments.
	for _, d := range definitionMap {
		for _, f := range d.FieldDefinitions {
			if id, ok := f.Validations["metaobject_definition_id"]; ok {
				if defType, ok := referenceTypes[id.(string)]; ok {
					f.Validations["metaobject_definition"] = defType
					delete(f.Validations, "metaobject_definition_id")
				}
			}

			if idsValue, ok := f.Validations["metaobject_definition_ids"]; ok {
				ids := idsValue.([]any)

				defTypes := make([]string, len(ids))
				for i, id := range ids {
					if defType, ok := referenceTypes[id.(string)]; ok {
						defTypes[i] = defType
					}
				}

				f.Validations["metaobject_definitions"] = defTypes
				delete(f.Validations, "metaobject_definition_ids")
			}
		}
	}

	return definitionMap
}
