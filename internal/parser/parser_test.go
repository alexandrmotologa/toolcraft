package parser_test

import (
	"path/filepath"
	"testing"

	"github.com/alexandrmotologa/toolcraft/internal/parser"
)

func TestDetectAndParseOpenAPI(t *testing.T) {
	path := filepath.Join("..", "..", "examples", "petstore_openapi3.yaml")
	spec, err := parser.DetectAndParse(path)
	if err != nil {
		t.Fatalf("failed to parse OpenAPI spec: %v", err)
	}

	if spec.Title != "Petstore Service API" {
		t.Errorf("expected title 'Petstore Service API', got '%s'", spec.Title)
	}

	if len(spec.Tools) < 4 {
		t.Errorf("expected at least 4 tools, got %d", len(spec.Tools))
	}

	// Verify specific tools
	var listPetsFound, getPetFound, createPetFound, deletePetFound bool
	for _, tool := range spec.Tools {
		if tool.OriginalID == "listPets" {
			listPetsFound = true
			if !tool.ReadOnly {
				t.Errorf("expected listPets to be readOnly")
			}
			if tool.Method != "GET" {
				t.Errorf("expected GET method, got %s", tool.Method)
			}
		}
		if tool.OriginalID == "getPetById" {
			getPetFound = true
			if len(tool.Parameters) < 2 {
				t.Errorf("expected at least 2 parameters for getPetById, got %d", len(tool.Parameters))
			}
		}
		if tool.OriginalID == "createPet" {
			createPetFound = true
			if tool.ReadOnly {
				t.Errorf("expected createPet to NOT be readOnly")
			}
			if tool.RequestBody == nil {
				t.Errorf("expected requestBody for createPet")
			}
		}
		if tool.OriginalID == "deletePet" {
			deletePetFound = true
			if tool.ReadOnly {
				t.Errorf("expected deletePet to NOT be readOnly")
			}
		}
	}

	if !listPetsFound || !getPetFound || !createPetFound || !deletePetFound {
		t.Errorf("missing expected tools: listPets=%v, getPet=%v, createPet=%v, deletePet=%v",
			listPetsFound, getPetFound, createPetFound, deletePetFound)
	}
}

func TestDetectAndParseSwagger(t *testing.T) {
	path := filepath.Join("..", "..", "examples", "swagger_v2.json")
	spec, err := parser.DetectAndParse(path)
	if err != nil {
		t.Fatalf("failed to parse Swagger spec: %v", err)
	}

	if spec.Title != "Inventory Management API" {
		t.Errorf("expected title 'Inventory Management API', got '%s'", spec.Title)
	}

	if len(spec.Tools) != 2 {
		t.Errorf("expected 2 tools, got %d", len(spec.Tools))
	}
}

func TestDetectAndParsePostman(t *testing.T) {
	path := filepath.Join("..", "..", "examples", "postman_collection_v2.json")
	spec, err := parser.DetectAndParse(path)
	if err != nil {
		t.Fatalf("failed to parse Postman spec: %v", err)
	}

	if spec.Title != "Billing and Customer Collection" {
		t.Errorf("expected title 'Billing and Customer Collection', got '%s'", spec.Title)
	}

	if len(spec.Tools) != 2 {
		t.Errorf("expected 2 tools, got %d", len(spec.Tools))
	}
}

func TestDetectAndParseCurl(t *testing.T) {
	path := filepath.Join("..", "..", "examples", "curl_sample.txt")
	spec, err := parser.DetectAndParse(path)
	if err != nil {
		t.Fatalf("failed to parse cURL spec: %v", err)
	}

	if len(spec.Tools) != 2 {
		t.Fatalf("expected 2 tools from cURL snippet, got %d", len(spec.Tools))
	}

	if spec.Tools[0].Method != "GET" {
		t.Errorf("expected first tool to be GET, got %s", spec.Tools[0].Method)
	}
	if spec.Tools[1].Method != "POST" {
		t.Errorf("expected second tool to be POST, got %s", spec.Tools[1].Method)
	}
}
