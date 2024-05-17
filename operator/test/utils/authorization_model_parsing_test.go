package utils

import (
	"github.com/openfga/language/pkg/go/transformer"
	"strings"
	"testing"
)

func TestGivenDslThenParseToJson(t *testing.T) {
	// Arrange
	dslSchema := `
model
  schema 1.1

type user

type document
  relations
    define reader: [user]
    define writer: [user]
    define owner: [user]
`
	expectedJson := `{
    "schema_version": "1.1",
    "type_definitions": [
        {
            "type": "user"
        },
        {
            "type": "document",
            "relations": {
                "owner": {
                    "this": {}
                },
                "reader": {
                    "this": {}
                },
                "writer": {
                    "this": {}
                }
            },
            "metadata": {
                "relations": {
                    "owner": {
                        "directly_related_user_types": [
                            {
                                "type": "user"
                            }
                        ]
                    },
                    "reader": {
                        "directly_related_user_types": [
                            {
                                "type": "user"
                            }
                        ]
                    },
                    "writer": {
                        "directly_related_user_types": [
                            {
                                "type": "user"
                            }
                        ]
                    }
                }
            }
        }
    ]
}`

	// Act
	generatedJsonString, err := transformer.TransformDSLToJSON(dslSchema)

	// Assert
	if err != nil {
		t.Fatalf("Error transforming DSL to JSON: %v", err)
	}
	expectedReplaced := replaceAllWhiteSpace(expectedJson)
	actualReplaced := replaceAllWhiteSpace(generatedJsonString)
	if actualReplaced != expectedReplaced {
		t.Errorf("Mismatch in JSON output:\nExpected:\n%s\nActual:\n%s", expectedReplaced, actualReplaced)
	}
}

func replaceAllWhiteSpace(value string) string {
	return strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(value, "\n", ""), "\t", ""), " ", "")
}
