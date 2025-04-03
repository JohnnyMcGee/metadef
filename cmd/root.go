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
	"github.com/sergi/go-diff/diffmatchpatch"
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
	rootCmd.PersistentFlags().StringVarP(&shop, "shop", "s", "", "Shopify shop domain (without the .myshopify.com extension)")
	rootCmd.MarkFlagRequired("shop")
	rootCmd.PersistentFlags().StringVarP(&outFile, "out", "o", "", "Output file name")

	diffCmd.MarkFlagRequired("file")

	rootCmd.AddCommand(pullCmd)
	rootCmd.AddCommand(diffCmd)
}

var diffCmd = &cobra.Command{
	Use:   "diff <file or directory>",
	Short: "Compare local metaobject definitions with the Shopify store",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		config, err := ReadConfig(configFile)
		if err != nil {
			log.Fatalf("Error reading config: %v\n", err)
			return err
		}

		client := shopify.NewShopifyAdminClient(shop, config.Shops[shop], config.Version)

		input, err := os.ReadFile(args[0])
		if err != nil {
			log.Fatalf("Error reading local definitions: %v\n", err)
		}

		var inputDefinitions map[string]core.MetaobjectDefinition
		hjson.Unmarshal(input, &inputDefinitions)

		for key, localDefinition := range inputDefinitions {
			result, err := shopify.GetMetaobjectDefinitionByType(context.Background(), client, key)

			if err != nil {
				log.Fatalf("Error fetching remote definition for %v: %v\n", key, err)
				return err
			}

			remoteDefinition := core.ConvertMetaobjectDefinition(result.MetaobjectDefinitionByType)

			localJson, err := hjson.Marshal(localDefinition)
			if err != nil {
				log.Fatalf("Error marshalling local definition %v: %v\n", key, err)
				return err
			}

			remoteJson, err := hjson.Marshal(remoteDefinition)
			if err != nil {
				log.Fatalf("Error marshalling remote definition %v: %v\n", key, err)
				return err
			}

			dmp := diffmatchpatch.New()

			match := dmp.MatchMain(string(remoteJson), string(localJson), 0)

			if match == 0 {
				continue
			}

			diffs := dmp.DiffMain(string(remoteJson), string(localJson), false)
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
			fmt.Printf("%s: \x1b[32m+%d\x1b[0m \x1b[31m-%d\x1b[0m\n", result.MetaobjectDefinitionByType.Name, inserts, deletes)
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

		referenceTypes := make(map[string]string)

		for _, definition := range data.MetaobjectDefinitions.Nodes {
			definitions[definition.Type] = core.ConvertMetaobjectDefinition(definition)
			referenceTypes[definition.Id] = definition.Type
		}

		// Normalize metaobject definition references to use the type name
		// instead of the ID. This allows us to use the same definitions
		// across different stores and environments.
		for _, d := range definitions {
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

		payload, err := hjson.Marshal(definitions)
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
