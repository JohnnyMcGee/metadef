/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/JohnnyMcGee/metadef/core"
	"github.com/JohnnyMcGee/metadef/shopify"
	"github.com/spf13/cobra"

	"github.com/hjson/hjson-go/v4"
	"github.com/sergi/go-diff/diffmatchpatch"
)

const DEFAULT_API_VERSION = "2025-04"

var (
	shop       string
	outFile    string
	configFile string
	config     Config
)

type Config struct {
	Shops   map[string]string `hjson:"shops"`
	Version string            `hjson:"version"`
}

func ReadConfig(path string) (Config, error) {

	b, err := os.ReadFile(path)
	if err != nil {
		return Config{}, errors.New("Error reading config file: " + path)
	}

	var config Config
	if err := hjson.Unmarshal(b, &config); err != nil {
		return Config{}, errors.New("Error unmarshalling config file: " + path)
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
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "Config file path")
	rootCmd.PersistentFlags().StringVarP(&shop, "shop", "s", "", "Shopify shop domain (without the .myshopify.com extension)")
	rootCmd.PersistentFlags().StringVarP(&outFile, "out", "o", "", "Output file name")

	rootCmd.AddCommand(pullCmd)
	rootCmd.AddCommand(diffCmd)
	rootCmd.AddCommand(pushCmd)
}

func initDefaults() {
	if configFile == "" {
		configFile = os.Getenv("HOME") + "/.metadef.hjson"
	}

	var err error
	config, err = ReadConfig(configFile)
	if err != nil {
		log.Fatalf("Error reading config: %v\n", err)
	}

	if len(config.Shops) == 0 {
		log.Fatalf("Error: No shops found in config file: %s\n", configFile)
	}

	if shop == "" {
		for key := range config.Shops {
			shop = key
			break
		}
	}

	if config.Version == "" {
		config.Version = DEFAULT_API_VERSION
	}
}

var pushCmd = &cobra.Command{
	Use:   "push <file or directory>",
	Short: "Push local metaobject definitions to the Shopify store",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		initDefaults()
		log.Printf("Using config file %s\n", configFile)
		log.Printf("Pushing definitions from file %s to shop %s\n", args[0], shop)
		client := shopify.NewShopifyAdminClient(shop, config.Shops[shop], config.Version)
		ms := &core.MetaobjectService{ShopifyClient: &client}

		input, err := os.ReadFile(args[0])
		if err != nil {
			log.Fatalf("Error reading local definitions: %v\n", err)
		}

		var inputDefinitions map[string]core.MetaobjectDefinition
		hjson.Unmarshal(input, &inputDefinitions)

		return ms.Push(inputDefinitions)
	},
}

var diffCmd = &cobra.Command{
	Use:   "diff <file or directory>",
	Short: "Compare local metaobject definitions with the Shopify store",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		initDefaults()
		log.Printf("Using config file %s\n", configFile)
		log.Printf("Diffing metaobject definitions from file %s to shop %s\n", args[0], shop)

		client := shopify.NewShopifyAdminClient(shop, config.Shops[shop], config.Version)
		ms := &core.MetaobjectService{ShopifyClient: &client}

		input, err := os.ReadFile(args[0])
		if err != nil {
			log.Fatalf("Error reading local definitions: %v\n", err)
		}

		var inputDefinitions map[string]core.MetaobjectDefinition
		hjson.Unmarshal(input, &inputDefinitions)

		diffMap, err := ms.Diff(inputDefinitions)
		if err != nil {
			log.Fatalf("Error diffing definitions: %v\n", err)
			return err
		}

		dmp := diffmatchpatch.New()

		for key, diffs := range diffMap {
			inserts := 0
			deletes := 0

			for _, diff := range diffs {
				if diff.Type == diffmatchpatch.DiffInsert {
					inserts += len(diff.Text)
				} else if diff.Type == diffmatchpatch.DiffDelete {
					deletes += len(diff.Text)
				}
			}

			fmt.Println()
			fmt.Printf("%s: \x1b[32m+%d\x1b[0m \x1b[31m-%d\x1b[0m\n", key, inserts, deletes)
			fmt.Println("---------------------------------")
			fmt.Println(dmp.DiffPrettyText(diffs))
		}

		return nil
	},
}

var pullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Pull all metaobject definitions from the Shopify store",
	RunE: func(cmd *cobra.Command, args []string) error {
		initDefaults()
		log.Printf("Using config file %s\n", configFile)
		log.Printf("Pulling metaobject definitions from shop %s\n", shop)

		client := shopify.NewShopifyAdminClient(shop, config.Shops[shop], config.Version)
		ms := &core.MetaobjectService{ShopifyClient: &client}

		defs, err := ms.Pull()
		if err != nil {
			log.Fatalf("Error pulling definitions: %v\n", err)
			return err
		}

		payload, err := hjson.Marshal(defs)
		if err != nil {
			log.Fatalf("Error marshalling data: %v\n", err)
			return err
		}

		if outFile != "" {
			os.WriteFile(outFile, payload, 0644)
		} else {
			log.Printf("%s\n", payload)
		}

		return nil
	},
}
