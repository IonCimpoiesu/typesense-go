package main

import (
	"io"
	"log"
	"net/http"
	"os"

	"gopkg.in/yaml.v3"
)

type yml map[string]interface{}

const (
	query = "query"
	array = "array"
)

// This script makes the changes needed for oapi-codegen to generate client_gen.go and types_gen.go from
// https://github.com/typesense/typesense-api-spec/blob/master/openapi.yml

func main() {
	m := make(yml)

	log.Println("Fetching openapi.yml from typesense api spec")
	err := fetchOpenAPISpec()
	if err != nil {
		log.Fatalf("Aboring: %s", err.Error())
	}

	configFile, err := os.Open("./typesense/api/generator/openapi.yml")
	if err != nil {
		log.Fatalf("Unable to open config file: %s", err.Error())
		return
	}

	decoder := yaml.NewDecoder(configFile)
	err = decoder.Decode(&m)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	// Unwrapping the search parameters
	log.Println("Unwrapping search parameters")
	unwrapSearchParameters(&m)
	// Unwrapping import and export parameters
	log.Println("Unwrapping documents import parameters")
	unwrapImportDocuments(&m)
	log.Println("Unwrapping documents export parameters")
	unwrapExportDocuments(&m)
	// Unwrapping delete document parameters
	log.Println("Unwrapping documents delete parameters")
	unwrapDeleteDocument(&m)
	// Remove additionalProperties from SearchResultHit -> document
	log.Println("Removing additionalProperties from SearchResultHit")
	searchResultHit(&m)

	log.Println("Writing updated spec to generator.yml")
	generatorFile, err := os.Create("./typesense/api/generator/generator.yml")
	if err != nil {
		log.Fatalf("Unable to open config file: %s", err.Error())
		return
	}

	encode := yaml.NewEncoder(generatorFile)
	encode.SetIndent(2)
	err = encode.Encode(m)
	if err != nil {
		log.Fatalf("error: %v", err)
	}
}

func fetchOpenAPISpec() error {
	url := "https://raw.githubusercontent.com/typesense/typesense-api-spec/master/openapi.yml"

	// Fetch the spec
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Unable to fetch spec: %s", err.Error())
		return nil
	}
	defer resp.Body.Close()

	// Write the spec to openapi.yml file
	openapiFile, err := os.Create("./typesense/api/generator/openapi.yml")
	if err != nil {
		log.Printf("Unable to write openapi.yml file: %s", err.Error())
		return nil
	}
	defer openapiFile.Close()

	// Write the body to openapi.yml file
	_, err = io.Copy(openapiFile, resp.Body)
	return err
}

func searchResultHit(m *yml) {
	properties := (*m)["components"].(yml)["schemas"].(yml)["SearchResultHit"].(yml)["properties"].(yml)
	document := properties["document"].(yml)
	delete(document, "additionalProperties")
}

func unwrapDeleteDocument(m *yml) {
	parameters := (*m)["paths"].(yml)["/collections/{collectionName}/documents"].(yml)["delete"].(yml)["parameters"].([]interface{})
	deleteParameters := parameters[1].(yml)["schema"].(yml)["properties"].(yml)
	for k, v := range deleteParameters {
		newMap := make(yml)
		newMap["name"] = k
		newMap["in"] = query
		newMap["schema"] = make(yml)
		newMap["schema"].(yml)["type"] = v.(yml)["type"].(string)
		parameters = append(parameters, newMap)
	}
	parameters = append(parameters[:1], parameters[2:]...)
	(*m)["paths"].(yml)["/collections/{collectionName}/documents"].(yml)["delete"].(yml)["parameters"] = parameters
}

func unwrapExportDocuments(m *yml) {
	parameters := (*m)["paths"].(yml)["/collections/{collectionName}/documents/export"].(yml)["get"].(yml)["parameters"].([]interface{})
	exportParameters := parameters[1].(yml)["schema"].(yml)["properties"].(yml)
	for k, v := range exportParameters {
		newMap := make(yml)
		newMap["name"] = k
		newMap["in"] = query
		newMap["schema"] = make(yml)
		if v.(yml)["type"].(string) == array {
			newMap["schema"].(yml)["type"] = array
			newMap["schema"].(yml)["items"] = v.(yml)["items"]
		} else {
			newMap["schema"].(yml)["type"] = v.(yml)["type"].(string)
		}
		parameters = append(parameters, newMap)
	}
	parameters = append(parameters[:1], parameters[2:]...)
	(*m)["paths"].(yml)["/collections/{collectionName}/documents/export"].(yml)["get"].(yml)["parameters"] = parameters
}

func unwrapImportDocuments(m *yml) {
	parameters := (*m)["paths"].(yml)["/collections/{collectionName}/documents/import"].(yml)["post"].(yml)["parameters"].([]interface{})
	importParameters := parameters[1].(yml)["schema"].(yml)["properties"].(yml)

	for k, v := range importParameters {
		newMap := make(yml)
		newMap["name"] = k
		newMap["in"] = query
		newMap["schema"] = make(yml)
		newMap["schema"].(yml)["type"] = v.(yml)["type"].(string)
		if v.(yml)["enum"] != nil {
			newMap["schema"].(yml)["enum"] = v.(yml)["enum"]
		}
		parameters = append(parameters, newMap)
	}
	parameters = append(parameters[:1], parameters[2:]...)
	(*m)["paths"].(yml)["/collections/{collectionName}/documents/import"].(yml)["post"].(yml)["parameters"] = parameters
}

func unwrapSearchParameters(m *yml) {
	parameters := (*m)["paths"].(yml)["/collections/{collectionName}/documents/search"].(yml)["get"].(yml)["parameters"].([]interface{})
	searchParameters := parameters[1].(yml)["schema"].(yml)["properties"].(yml)

	for k, v := range searchParameters {
		newMap := make(yml)
		newMap["name"] = k
		if k == "q" || k == "query_by" {
			newMap["required"] = true
		}
		newMap["in"] = query
		newMap["schema"] = make(yml)
		if v.(yml)["oneOf"] == nil {
			if v.(yml)["type"].(string) == array {
				newMap["schema"].(yml)["type"] = array
				newMap["schema"].(yml)["items"] = v.(yml)["items"]
			} else {
				newMap["schema"].(yml)["type"] = v.(yml)["type"].(string)
			}
		} else {
			newMap["schema"].(yml)["oneOf"] = v.(yml)["oneOf"]
		}
		parameters = append(parameters, newMap)
	}

	parameters = append(parameters[:1], parameters[2:]...)
	(*m)["paths"].(yml)["/collections/{collectionName}/documents/search"].(yml)["get"].(yml)["parameters"] = parameters
}
