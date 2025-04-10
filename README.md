# Usage
## Install
```sh
go install github.com/JohnnyMcGee/metadef
```

## Config
Configuration is done using a hjson file. By default, the CLI looks for a global config file at `$HOME/.metadef.hjson`. You can pass a local config file using the flag `metadef -c /path/to/config.hjson`.

```hjson
{
// Map shop domain to admin access token. There must be at least one shop configured.
  shops: {
    shop-domain-1: shopify-admin-api-access-token-1
    shop-domain-2: shopify-admin-api-access-token-2
    shop-domain-3: shopify-admin-api-access-token-3
  }
// Optionally specify the Shopify Admin API version to use.
  version: 2025-04
}
```

### `shops`
`shops` is a map from shop domain to admin api access token. This enables the CLI to access each of your shops.

#### Shop Domain
The CLI uses Shopify Admin GraphQL API. To gain access to your store, add an entry to the `shops` map. The key should be your myshopify domain (without the .myshopify.com suffix). For example, if your store is `super-awesome-store.myshopify.com`, the key will be `super-awesome-store`.

#### Access Token
The easiest way to get an admin access token is to [create a custom shopify app](https://help.shopify.com/en/manual/apps/app-types/custom-apps).

Your app will need read/write permissions for metaobject definitions and metaobjects.

# Development
## Update GraphQL Schema
Periodically it may be necessary to fetch the latest schema for Shopify Admin GraphQL API. One way to do this is using the [get-graphql-schema CLI tool](https://github.com/gqlgo/get-graphql-schema).

1. Install get-grapqhl-schema tool
```sh
go install github.com/gqlgo/get-graphql-schema@latest
```

2. Download the schema from Shopify's public endpoint
```sh
get-graphql-schema https://shopify.dev/admin-graphql-direct-proxy/2025-04 > schema.graphql
```
Replace '2025-04' with the desired API version. API Versions are listed at https://shopify.dev/docs/api/admin-graphql