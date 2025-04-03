package core

import (
	"encoding/json"
	"errors"
	"log"
	"strings"

	"github.com/JohnnyMcGee/metaobjects-cli/shopify"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type FieldDefinition struct {
	Type        string         `json:"type"`
	Name        string         `json:"name,omitempty"`
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
		Name:        definition.Name,
		Description: definition.Description,
		Required:    definition.Required,
	}

	if definition.Name == titleCase(definition.Key) {
		f.Name = ""
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

func titleCase(s string) string {
	c := cases.Title(language.English)
	return c.String(strings.ReplaceAll(strings.ReplaceAll(s, "_", " "), "-", " "))
}

func ConvertMetaobjectDefinition(definition shopify.Cli_MetaobjectDefinition) MetaobjectDefinition {
	d := MetaobjectDefinition{
		Name:             definition.Name,
		Description:      definition.Description,
		DisplayNameKey:   definition.DisplayNameKey,
		FieldDefinitions: make(map[string]FieldDefinition, len(definition.FieldDefinitions)),
	}

	if definition.Name == titleCase(definition.Type) {
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

func NewMetaobjectFieldValidations(validations map[string]any, referenceIds map[string]string) ([]shopify.MetafieldDefinitionValidationInput, error) {
	fieldValidations := make([]shopify.MetafieldDefinitionValidationInput, 0, len(validations))

	if len(validations) == 0 {
		return fieldValidations, nil
	}

	for k, v := range validations {

		if k == "metaobject_definition" {
			id, ok := referenceIds[v.(string)]
			if !ok {
				log.Fatalf("Metaobject definition %s not found in reference IDs\n", v.(string))
				return nil, errors.New("metaobject definition not found")
			}

			fieldValidations = append(fieldValidations, shopify.MetafieldDefinitionValidationInput{
				Name:  k,
				Value: id,
			})

			continue
		}

		if k == "metaobject_definitions" {
			ids, ok := v.([]any)
			if !ok {
				log.Fatalf("Metaobject definitions %v not found in reference IDs\n", v)
				return nil, errors.New("metaobject definitions not found")
			}

			definitions := make([]string, len(ids))
			for i, id := range ids {
				if referenceId, ok := referenceIds[id.(string)]; ok {
					definitions[i] = referenceId
				}
			}

			value, err := json.Marshal(definitions)
			if err != nil {
				log.Fatalf("Error marshalling validation value for field %s: %v\n", k, err)
				return nil, err
			}

			fieldValidations = append(fieldValidations, shopify.MetafieldDefinitionValidationInput{
				Name:  k,
				Value: string(value),
			})

			continue
		}

		valueJson, err := json.Marshal(v)

		if err != nil {
			log.Fatalf("Error marshalling validation value: %s, %v\n", k, v)
			return nil, err
		}

		fieldValidations = append(fieldValidations, shopify.MetafieldDefinitionValidationInput{
			Name:  k,
			Value: string(valueJson),
		})
	}

	return fieldValidations, nil
}

func NewMetaobjectFieldCreateInput(key string, field FieldDefinition, referenceIds map[string]string) (shopify.MetaobjectFieldDefinitionCreateInput, error) {
	input := shopify.MetaobjectFieldDefinitionCreateInput{
		Key:         key,
		Type:        field.Type,
		Name:        field.Name,
		Description: field.Description,
		Required:    field.Required,
	}

	if input.Name == "" {
		name := titleCase(key)
		input.Name = name
	}

	validations, err := NewMetaobjectFieldValidations(field.Validations, referenceIds)
	if err != nil {
		log.Fatalf("Error creating field validations for field %s: %v\n", key, err)
		return shopify.MetaobjectFieldDefinitionCreateInput{}, err
	}

	input.Validations = validations

	return input, nil
}

func NewMetaobjectDefinitionCreateInput(defType string, definition MetaobjectDefinition, referenceIds map[string]string) (shopify.MetaobjectDefinitionCreateInput, error) {
	publicRead := shopify.MetaobjectStorefrontAccessPublicRead

	input := shopify.MetaobjectDefinitionCreateInput{
		Type: defType,
		Access: shopify.CustomMetaobjectAccessInput{
			Storefront: publicRead,
		},
		Name:             definition.Name,
		Description:      definition.Description,
		FieldDefinitions: make([]shopify.MetaobjectFieldDefinitionCreateInput, 0, len(definition.FieldDefinitions)),
		DisplayNameKey:   definition.DisplayNameKey,
	}

	if definition.Name == "" {
		name := titleCase(defType)
		input.Name = name
	}

	if definition.Access != nil {
		if definition.Access.Storefront != "" {
			input.Access.Storefront = definition.Access.Storefront
		}

		switch definition.Access.Admin {
		case shopify.MetaobjectAdminAccessMerchantRead:
			input.Access.Admin = shopify.MetaobjectAdminAccessInputMerchantRead
		case shopify.MetaobjectAdminAccessMerchantReadWrite:
			input.Access.Admin = shopify.MetaobjectAdminAccessInputMerchantReadWrite
		}
	}

	if definition.Capabilities != nil {
		input.Capabilities = shopify.MetaobjectCapabilityCreateInput{
			Publishable: shopify.MetaobjectCapabilityPublishableInput{
				Enabled: definition.Capabilities.Publishable,
			},
			Translatable: shopify.MetaobjectCapabilityTranslatableInput{
				Enabled: definition.Capabilities.Translatable,
			},
		}

		if onlineStore := definition.Capabilities.OnlineStore; onlineStore != nil {
			input.Capabilities.OnlineStore = shopify.MetaobjectCapabilityOnlineStoreInput{
				Enabled: true,
				Data: shopify.MetaobjectCapabilityDefinitionDataOnlineStoreInput{
					CreateRedirects: onlineStore.CanCreateRedirects,
					UrlHandle:       onlineStore.UrlHandle,
				},
			}
		}

		if renderable := definition.Capabilities.Renderable; renderable != nil {
			input.Capabilities.Renderable = shopify.MetaobjectCapabilityRenderableInput{
				Enabled: true,
				Data: shopify.MetaobjectCapabilityDefinitionDataRenderableInput{
					MetaDescriptionKey: renderable.MetaDescriptionKey,
					MetaTitleKey:       renderable.MetaTitleKey,
				},
			}
		}
	}

	for key, field := range definition.FieldDefinitions {
		if input.DisplayNameKey == "" && field.Type == "single_line_text_field" {
			input.DisplayNameKey = key
		}

		fieldDefinition, err := NewMetaobjectFieldCreateInput(key, field, referenceIds)
		if err != nil {
			log.Fatalf("Error creating field input for field %s: %v\n", key, err)
			return shopify.MetaobjectDefinitionCreateInput{}, err
		}

		input.FieldDefinitions = append(input.FieldDefinitions, fieldDefinition)
	}

	return input, nil
}

func NewMetaobjectDefinitionUpdateInput(defType string, definition MetaobjectDefinition, prevDefinition MetaobjectDefinition, referenceIds map[string]string) (shopify.MetaobjectDefinitionUpdateInput, error) {
	input := shopify.MetaobjectDefinitionUpdateInput{
		Access: shopify.CustomMetaobjectAccessInput{
			Storefront: shopify.MetaobjectStorefrontAccessPublicRead,
		},
		Name:             definition.Name,
		Description:      definition.Description,
		FieldDefinitions: make([]shopify.CustomMetaobjectFieldDefinitionOperationInput, 0, len(definition.FieldDefinitions)),
		DisplayNameKey:   definition.DisplayNameKey,
	}

	if definition.Name == "" {
		name := titleCase(defType)
		input.Name = name
	}

	if definition.Access != nil {
		if definition.Access.Storefront != "" {
			input.Access.Storefront = definition.Access.Storefront
		}

		switch definition.Access.Admin {
		case shopify.MetaobjectAdminAccessMerchantRead:
			input.Access.Admin = shopify.MetaobjectAdminAccessInputMerchantRead
		case shopify.MetaobjectAdminAccessMerchantReadWrite:
			input.Access.Admin = shopify.MetaobjectAdminAccessInputMerchantReadWrite
		}
	}

	if definition.Capabilities != nil {
		input.Capabilities = shopify.MetaobjectCapabilityUpdateInput{
			Publishable: shopify.MetaobjectCapabilityPublishableInput{
				Enabled: definition.Capabilities.Publishable,
			},
			Translatable: shopify.MetaobjectCapabilityTranslatableInput{
				Enabled: definition.Capabilities.Translatable,
			},
		}

		if onlineStore := definition.Capabilities.OnlineStore; onlineStore != nil {
			input.Capabilities.OnlineStore = shopify.MetaobjectCapabilityOnlineStoreInput{
				Enabled: true,
				Data: shopify.MetaobjectCapabilityDefinitionDataOnlineStoreInput{
					CreateRedirects: onlineStore.CanCreateRedirects,
					UrlHandle:       onlineStore.UrlHandle,
				},
			}
		}

		if renderable := definition.Capabilities.Renderable; renderable != nil {
			input.Capabilities.Renderable = shopify.MetaobjectCapabilityRenderableInput{
				Enabled: true,
				Data: shopify.MetaobjectCapabilityDefinitionDataRenderableInput{
					MetaDescriptionKey: renderable.MetaDescriptionKey,
					MetaTitleKey:       renderable.MetaTitleKey,
				},
			}
		}
	}

	for key, field := range definition.FieldDefinitions {
		if _, ok := prevDefinition.FieldDefinitions[key]; !ok {

			create, err := NewMetaobjectFieldCreateInput(key, field, referenceIds)
			if err != nil {
				log.Fatalf("Error creating field input for field %s: %v\n", key, err)
				return shopify.MetaobjectDefinitionUpdateInput{}, err
			}

			input.FieldDefinitions = append(input.FieldDefinitions, shopify.CustomMetaobjectFieldDefinitionOperationInput{
				Create: &create,
			})

			continue
		}

		update := shopify.MetaobjectFieldDefinitionUpdateInput{
			Key:         key,
			Name:        field.Name,
			Description: field.Description,
			Required:    field.Required,
		}

		if field.Name == "" {
			update.Name = titleCase(key)
		}

		if input.DisplayNameKey == "" && field.Type == "single_line_text_field" {
			input.DisplayNameKey = key
		}

		validations, err := NewMetaobjectFieldValidations(field.Validations, referenceIds)
		if err != nil {
			log.Fatalf("Error creating field validations for field %s: %v\n", key, err)
			return shopify.MetaobjectDefinitionUpdateInput{}, err
		}
		update.Validations = validations

		input.FieldDefinitions = append(input.FieldDefinitions, shopify.CustomMetaobjectFieldDefinitionOperationInput{Update: &update})

	}

	for key := range prevDefinition.FieldDefinitions {
		if _, ok := definition.FieldDefinitions[key]; ok {
			continue
		}

		del := shopify.MetaobjectFieldDefinitionDeleteInput{
			Key: key,
		}

		input.FieldDefinitions = append(input.FieldDefinitions, shopify.CustomMetaobjectFieldDefinitionOperationInput{
			Delete: &del,
		})
	}

	return input, nil
}
