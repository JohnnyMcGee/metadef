package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/Khan/genqlient/graphql"
	"github.com/hjson/hjson-go/v4"
)

type ShopConfig struct {
	Domain string
	Token  string
}

type Config struct {
	Shops map[string]ShopConfig
}

const ConfigFile = ".metaobjectsrc.hjson"

func ReadConfig(FilePath string) (Config, error) {
	b, err := os.ReadFile(ConfigFile)
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

func main() {
	config, err := ReadConfig(ConfigFile)
	if err != nil {
		fmt.Printf("Error reading config: %v\n", err)
		return
	}

	client := graphql.NewClient(fmt.Sprintf("https://%s/admin/api/2025-04/graphql.json", config.Shops["cowabunga"].Domain), &http.Client{Transport: &authedTransport{token: config.Shops["cowabunga"].Token, wrapped: http.DefaultTransport}})
	data, err := ListMetaobjectDefinitions(context.Background(), client, 250, "")
	if err != nil {
		fmt.Printf("Error fetching data: %v\n", err)
		return
	}

	fmt.Printf("config: %v\n", config)
	fmt.Printf("data: %v\n", data)
}
