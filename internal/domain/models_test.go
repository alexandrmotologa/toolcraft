package domain_test

import (
	"testing"

	"github.com/alexandrmotologa/toolcraft/internal/domain"
)

func TestSanitizeToolID(t *testing.T) {
	tests := []struct {
		opID   string
		method string
		path   string
		want   string
	}{
		{"listPets", "GET", "/pets", "list_pets"},
		{"getPetById", "GET", "/pets/{petId}", "get_pet_by_id"},
		{"", "GET", "/api/v1/customers/{id}/invoices", "get_api_v1_customers_id_invoices"},
		{"create-item", "POST", "/items", "create_item"},
	}

	for _, tt := range tests {
		got := domain.SanitizeToolID(tt.opID, tt.method, tt.path)
		if got != tt.want {
			t.Errorf("SanitizeToolID(%q, %q, %q) = %q; want %q", tt.opID, tt.method, tt.path, got, tt.want)
		}
	}
}
