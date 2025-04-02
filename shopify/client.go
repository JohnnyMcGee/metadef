package shopify

import (
	"fmt"
	"net/http"

	"github.com/Khan/genqlient/graphql"
)

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
