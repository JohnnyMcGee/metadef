package core

import (
	"context"
	"errors"
	"log"

	"github.com/JohnnyMcGee/metadef/shopify"
	"github.com/Khan/genqlient/graphql"
	"github.com/hjson/hjson-go"
	"github.com/sergi/go-diff/diffmatchpatch"
)

type MetaobjectService struct {
	ShopifyClient *graphql.Client
}

func (ms *MetaobjectService) Pull() (map[string]MetaobjectDefinition, error) {

	data, err := shopify.ListMetaobjectDefinitions(context.Background(), *ms.ShopifyClient, 250)
	if err != nil {
		log.Fatalf("Error fetching data: %v\n", err)
		return nil, err
	}

	return CreateMetaobjectDefinitionMap(data.MetaobjectDefinitions.Nodes), nil
}

func (ms *MetaobjectService) Diff(definitions map[string]MetaobjectDefinition) (map[string][]diffmatchpatch.Diff, error) {
	data, err := shopify.ListMetaobjectDefinitions(context.Background(), *ms.ShopifyClient, 250)
	if err != nil {
		log.Fatalf("Error Listing Metaobject Definitions: %v\n", err)
		return nil, err
	}

	remoteDefinitions := CreateMetaobjectDefinitionMap(data.MetaobjectDefinitions.Nodes)
	diffs := make(map[string][]diffmatchpatch.Diff)

	for key, localDefinition := range definitions {
		remoteDefinition := remoteDefinitions[key]

		localJson, err := hjson.Marshal(localDefinition)
		if err != nil {
			log.Fatalf("Error marshalling local definition %v: %v\n", key, err)
			return nil, err
		}

		remoteJson, err := hjson.Marshal(remoteDefinition)
		if err != nil {
			log.Fatalf("Error marshalling remote definition %v: %v\n", key, err)
			return nil, err
		}

		dmp := diffmatchpatch.New()

		match := dmp.MatchMain(string(remoteJson), string(localJson), 0)

		if match == 0 {
			continue
		}

		diffs[key] = dmp.DiffMain(string(remoteJson), string(localJson), false)
	}

	return diffs, nil
}

func (ms *MetaobjectService) Push(definitions map[string]MetaobjectDefinition) error {
	data, err := shopify.ListMetaobjectDefinitions(context.Background(), *ms.ShopifyClient, 250)
	if err != nil {
		log.Fatalf("Error fetching data: %v\n", err)
		return err
	}

	remoteDefinitions := CreateMetaobjectDefinitionMap(data.MetaobjectDefinitions.Nodes)

	referenceMap := make(map[string]string, len(remoteDefinitions))
	for _, def := range data.MetaobjectDefinitions.Nodes {
		referenceMap[def.Type] = def.Id
	}

	for key, localDefinition := range definitions {
		if _, ok := remoteDefinitions[key]; ok {
			continue
		}

		input, err := NewMetaobjectDefinitionCreateInput(key, localDefinition, referenceMap)
		if err != nil {
			log.Fatalf("Error creating input for definition %v: %v\n", key, err)
			return err
		}

		_, err = shopify.CreateMetaobjectDefinition(context.Background(), *ms.ShopifyClient, input)
		if err != nil {
			log.Fatalf("Error creating definition %v: %v\n", key, err)
			return err
		}

		log.Printf("Created definition: %s\n", key)
	}

	data, err = shopify.ListMetaobjectDefinitions(context.Background(), *ms.ShopifyClient, 250)
	if err != nil {
		log.Fatalf("Error fetching data: %v\n", err)
		return err
	}

	remoteDefinitions = CreateMetaobjectDefinitionMap(data.MetaobjectDefinitions.Nodes)

	for _, def := range data.MetaobjectDefinitions.Nodes {
		referenceMap[def.Type] = def.Id
	}

	for key, localDefinition := range definitions {
		remoteDefinition, ok := remoteDefinitions[key]
		if !ok {
			log.Fatalf("Error finding definition %v: %v\n", key, err)
			return errors.New("definition not found")
		}

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

		input, err := NewMetaobjectDefinitionUpdateInput(key, localDefinition, remoteDefinition, referenceMap)
		if err != nil {
			log.Fatalf("Error creating input for definition %v: %v\n", key, err)
			return err
		}

		id, ok := referenceMap[key]
		if !ok {
			log.Fatalf("Error finding ID for definition %v: %v\n", key, err)
			return errors.New("ID not found")
		}

		res, err := shopify.UpdateMetaobjectDefinition(context.Background(), *ms.ShopifyClient, id, input)
		if err != nil {
			log.Fatalf("Error updating definition %v: %v\n", key, err)
			return err
		}

		if len(res.MetaobjectDefinitionUpdate.UserErrors) > 0 {
			log.Fatalf("Error updating definition %v: %v\n", key, res.MetaobjectDefinitionUpdate.UserErrors)
			return errors.New("Error updating definition")
		}

		log.Printf("Updated definition: %s\n", key)
	}

	return nil
}
