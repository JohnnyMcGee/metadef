package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"strings"

	"github.com/Khan/genqlient/graphql"
	"github.com/hjson/hjson-go/v4"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

const configFile = ".metaobjectsrc.hjson"
const shop = "earth-breeze-hydrogen"
const outFile = "ebh-defs.hjson"

type Config struct {
	Shops   map[string]string `hjson:"shops"`
	Version string            `hjson:"version"`
}

func ReadConfig(path string) (Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	var config Config
	if err := hjson.Unmarshal(b, &config); err != nil {
		return Config{}, err
	}

	return config, nil
}

type authedTransport struct {
	token   string
	wrapped http.RoundTripper
}

func (t *authedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("X-Shopify-Access-Token", t.token)
	return t.wrapped.RoundTrip(req)
}

func NewShopifyAdminClient(shop string, token string, version string) graphql.Client {
	url := fmt.Sprintf("https://%s.myshopify.com/admin/api/%s/graphql.json", shop, version)
	httpClient := http.Client{Transport: &authedTransport{token: token, wrapped: http.DefaultTransport}}
	return graphql.NewClient(url, &httpClient)
}

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
	Admin      MetaobjectAdminAccess      `json:"admin,omitempty"`
	Storefront MetaobjectStorefrontAccess `json:"storefront,omitempty"`
}

type MetaobjectDefinition struct {
	Name             string                     `json:"name,omitempty"`
	Description      string                     `json:"description,omitempty"`
	Access           *Access                    `json:"access,omitempty"`
	Capabilities     *Capabilities              `json:"capabilities,omitempty"`
	DisplayNameKey   string                     `json:"displayNameKey,omitempty"`
	FieldDefinitions map[string]FieldDefinition `json:"fieldDefinitions"`
}

func titleCase(s string) string {
	c := cases.Title(language.English)
	return c.String(strings.ReplaceAll(strings.ReplaceAll(s, "_", " "), "-", " "))
}

func ConvertAccess(access ListMetaobjectDefinitionsMetaobjectDefinitionsMetaobjectDefinitionConnectionNodesMetaobjectDefinitionAccessMetaobjectAccess) (a *Access, empty bool) {
	a = &Access{}
	empty = true

	if access.Admin != MetaobjectAdminAccessPublicReadWrite {
		a.Admin, empty = access.Admin, false
	}

	if access.Storefront != MetaobjectStorefrontAccessPublicRead {
		a.Storefront, empty = access.Storefront, false
	}

	return a, empty
}

func ConvertFieldDefinition(definition ListMetaobjectDefinitionsMetaobjectDefinitionsMetaobjectDefinitionConnectionNodesMetaobjectDefinitionFieldDefinitionsMetaobjectFieldDefinition) FieldDefinition {
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

func ConvertCapabilities(capabilities ListMetaobjectDefinitionsMetaobjectDefinitionsMetaobjectDefinitionConnectionNodesMetaobjectDefinitionCapabilitiesMetaobjectCapabilities) (cap *Capabilities, empty bool) {
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

func ConvertMetaobjectDefinition(definition ListMetaobjectDefinitionsMetaobjectDefinitionsMetaobjectDefinitionConnectionNodesMetaobjectDefinition) MetaobjectDefinition {
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

func main() {
	config, err := ReadConfig(configFile)
	if err != nil {
		fmt.Printf("Error reading config: %v\n", err)
		return
	}

	client := NewShopifyAdminClient(shop, config.Shops[shop], config.Version)

	data, err := ListMetaobjectDefinitions(context.Background(), client, 250, "")
	if err != nil {
		fmt.Printf("Error fetching data: %v\n", err)
		return
	}

	definitions := make(map[string]MetaobjectDefinition, len(data.MetaobjectDefinitions.Nodes))
	fmt.Printf("Definitions Pulled: %d\n", len(data.MetaobjectDefinitions.Nodes))

	for _, definition := range data.MetaobjectDefinitions.Nodes {
		definitions[definition.Type] = ConvertMetaobjectDefinition(definition)
	}

	payload, err := hjson.Marshal(definitions)
	if err != nil {
		fmt.Printf("Error marshalling data: %v\n", err)
	}

	os.WriteFile(outFile, payload, 0644)
}
