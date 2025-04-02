# Update GraphQL Schema
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