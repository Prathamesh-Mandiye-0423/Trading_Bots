package decimal

import (
	"testing"
)

func TestDecimal_Mul(t *testing.T) {
	p := MustFromString("50000.00")
	q := MustFromString("1.12345678")
	result := p.Mul(q)

	expected := "56172.83900000"
	if result.String() != expected {
		t.Errorf("Expected %s, got %s", expected, result.String())
	}
}
