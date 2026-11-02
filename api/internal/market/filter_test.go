package market

import "testing"

func TestIsStockOrETF(t *testing.T) {
	keep := []string{"2330", "6488", "2881A", "0050", "00878", "006208", "01002T", "020029", "911868", "2887Z1"}
	drop := []string{"700022", "700126", "030123", "080123", "123", ""}
	for _, c := range keep {
		if !IsStockOrETF(c) {
			t.Errorf("%q should be kept", c)
		}
	}
	for _, c := range drop {
		if IsStockOrETF(c) {
			t.Errorf("%q should be dropped", c)
		}
	}
}
