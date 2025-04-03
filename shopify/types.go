package shopify

type CustomMetaobjectAccessInput struct {
	Admin      MetaobjectAdminAccessInput `json:"admin,omitempty"`
	Storefront MetaobjectStorefrontAccess `json:"storefront,omitempty"`
}

// Metaobject access permissions for the Admin API. When the metaobject is app-owned, the owning app always has
// full access.
type MetaobjectAdminAccessInput string

const (
	// The merchant has read-only access. No other apps have access.
	MetaobjectAdminAccessInputMerchantRead MetaobjectAdminAccessInput = "MERCHANT_READ"
	// The merchant has read and write access. No other apps have access.
	MetaobjectAdminAccessInputMerchantReadWrite MetaobjectAdminAccessInput = "MERCHANT_READ_WRITE"
)

type CustomMetaobjectFieldDefinitionOperationInput struct {
	// The input fields for creating a metaobject field definition.
	Create *MetaobjectFieldDefinitionCreateInput `json:"create,omitempty"`
	// The input fields for updating a metaobject field definition.
	Update *MetaobjectFieldDefinitionUpdateInput `json:"update,omitempty"`
	// The input fields for deleting a metaobject field definition.
	Delete *MetaobjectFieldDefinitionDeleteInput `json:"delete,omitempty"`
}

// The input fields for updating a metaobject field definition.
type MetaobjectFieldDefinitionUpdateInput struct {
	// The key of the field definition to update.
	Key string `json:"key"`
	// A human-readable name for the field.
	Name string `json:"name"`
	// An administrative description of the field.
	Description string `json:"description"`
	// Whether metaobjects require a saved value for the field.
	Required bool `json:"required"`
	// Custom validations that apply to values assigned to the field.
	Validations []MetafieldDefinitionValidationInput `json:"validations"`
}

// The input fields for deleting a metaobject field definition.
type MetaobjectFieldDefinitionDeleteInput struct {
	// The key of the field definition to delete.
	Key string `json:"key"`
}
