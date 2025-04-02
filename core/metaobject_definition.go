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

func ConvertAccess(access shopify.ListMetaobjectDefinitionsMetaobjectDefinitionsMetaobjectDefinitionConnectionNodesMetaobjectDefinitionAccessMetaobjectAccess) (a *Access, empty bool) {
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

func ConvertFieldDefinition(definition shopify.ListMetaobjectDefinitionsMetaobjectDefinitionsMetaobjectDefinitionConnectionNodesMetaobjectDefinitionFieldDefinitionsMetaobjectFieldDefinition) FieldDefinition {
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

func ConvertCapabilities(capabilities shopify.ListMetaobjectDefinitionsMetaobjectDefinitionsMetaobjectDefinitionConnectionNodesMetaobjectDefinitionCapabilitiesMetaobjectCapabilities) (cap *Capabilities, empty bool) {
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

func ConvertMetaobjectDefinition(definition shopify.ListMetaobjectDefinitionsMetaobjectDefinitionsMetaobjectDefinitionConnectionNodesMetaobjectDefinition) MetaobjectDefinition {
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

		d.FieldDefinitions[f.Key] = ConvertFieldDefinition(f)

		if defaultDisplayNameKey == "" && f.Type.Name == "single_line_text_field" {
			defaultDisplayNameKey = f.Key
		}
	}

	if definition.DisplayNameKey == defaultDisplayNameKey {
		d.DisplayNameKey = ""
	}

	if cap, empty := ConvertCapabilities(definition.Capabilities); !empty {
		d.Capabilities = cap
	}

	if access, empty := ConvertAccess(definition.Access); !empty {
		d.Access = access
	}

	return d
}
