/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/JohnnyMcGee/metaobjects-cli/core"
	"github.com/JohnnyMcGee/metaobjects-cli/shopify"
	"github.com/spf13/cobra"

	"github.com/hjson/hjson-go/v4"
)

const configFile = ".metaobjectsrc.hjson"

var (
	shop    string
	outFile string
)

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

var rootCmd = &cobra.Command{
	Use:   "mobdef",
	Short: "Automated Shopify Metaobject Definitions from local files",
	Long: `Define your Shopify metadata in local files and use this CLI to push and pull
them to and from your Shopify store. This allows you to version control your metadata,
sync it across stores, and define your data schemas alongside your code.
`,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	pullCmd.Flags().StringVarP(&shop, "shop", "s", "", "Shopify shop domain (without the .myshopify.com extension)")
	pullCmd.MarkFlagRequired("shop")
	pullCmd.Flags().StringVarP(&outFile, "out", "o", "", "Output file name")

	rootCmd.AddCommand(pullCmd)
}

var pullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Pull all metaobject definitions from the Shopify store",
	RunE: func(cmd *cobra.Command, args []string) error {
		config, err := ReadConfig(configFile)
		if err != nil {
			log.Fatalf("Error reading config: %v\n", err)
			return err
		}

		client := shopify.NewShopifyAdminClient(shop, config.Shops[shop], config.Version)

		data, err := shopify.ListMetaobjectDefinitions(context.Background(), client, 250, "")
		if err != nil {
			log.Fatalf("Error fetching data: %v\n", err)
			return err
		}

		definitions := make(map[string]core.MetaobjectDefinition, len(data.MetaobjectDefinitions.Nodes))

		for _, definition := range data.MetaobjectDefinitions.Nodes {
			definitions[definition.Type] = core.ConvertMetaobjectDefinition(definition)
		}

		payload, err := hjson.Marshal(definitions)
		if err != nil {
			log.Fatalf("Error marshalling data: %v\n", err)
			return err
		}

		if outFile != "" {
			os.WriteFile(outFile, payload, 0644)
		} else {
			fmt.Printf("%s\n", payload)
		}

		return nil
	},
}
