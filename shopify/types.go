package shopify

type CustomMetaobjectAccessInput struct {
	Admin      MetaobjectAdminAccessInput `json:"admin,omitempty"`
	Storefront MetaobjectStorefrontAccess `json:"storefront,omitempty"`
}
