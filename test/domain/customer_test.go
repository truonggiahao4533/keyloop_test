package domain_test

import "keyloop-test/internal/domain"

import "testing"

func TestCustomer_FullName(t *testing.T) {
	cust := &domain.Customer{
		FirstName: "Jane",
		LastName:  "Smith",
	}

	expected := "Jane Smith"
	if got := cust.FullName(); got != expected {
		t.Errorf("FullName() = %v, want %v", got, expected)
	}
}
