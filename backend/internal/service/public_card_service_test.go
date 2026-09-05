package service

import "testing"

func TestNormalizePublicCardCodePreservesStoredCase(t *testing.T) {
	const code = "de7b4aada24ed73c87797063947d5a59"
	if got := normalizePublicCardCode("  " + code + "  "); got != code {
		t.Fatalf("normalizePublicCardCode() = %q, want %q", got, code)
	}
}

func TestPromoConcurrency(t *testing.T) {
	tests := []struct {
		name string
		card *RedeemCode
		want int
	}{
		{name: "promo value", card: &RedeemCode{Type: RedeemTypePromo, ValidityDays: 3}, want: 3},
		{name: "promo default", card: &RedeemCode{Type: RedeemTypePromo}, want: 1},
		{name: "legacy balance", card: &RedeemCode{Type: RedeemTypeBalance, ValidityDays: 30}, want: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.card.PromoConcurrency(); got != tt.want {
				t.Fatalf("PromoConcurrency() = %d, want %d", got, tt.want)
			}
		})
	}
}
