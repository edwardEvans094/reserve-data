package blockchain

import "testing"

func TestGasOracle(t *testing.T) {
	t.Skip("Gas Oracle test requires external dependency, skipping")

	gso := NewGasOracle()
	if err := gso.runGasPricing(); err != nil {
		t.Fatal(err)
	}
	if gso.standard == 0 || gso.fast == 0 || gso.safeLow == 0 {
		t.Errorf(
			"Gas pricing is not fetched, standard: %g, fast: %g, safeLow: %g",
			gso.standard, gso.fast, gso.safeLow,
		)
	}
}
