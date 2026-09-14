package sqlparser_test

import (
	"reflect"
	"testing"

	"github.com/viant/sqlparser"
)

// Published callers can store/pass the original function, not only call it.
var identifierParts func(string) ([]string, error) = sqlparser.TableIdentifierParts

func TestTableIdentifierFunctionValueCompatibility(t *testing.T) {
	got, err := identifierParts(`[project.dataset.table]`)
	if err != nil || !reflect.DeepEqual(got, []string{"project.dataset.table"}) {
		t.Fatalf("default function changed: %v, %v", got, err)
	}
	parser := sqlparser.TableIdentifierParser{Product: "BigQuery"}
	var productParts func(string) ([]string, error) = parser.Parts
	got, err = productParts(`[project:dataset.table]`)
	if err != nil || !reflect.DeepEqual(got, []string{"project", "dataset", "table"}) {
		t.Fatalf("product method: %v, %v", got, err)
	}
}
