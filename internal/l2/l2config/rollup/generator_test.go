package rollup

import "testing"

func TestNormalizeHoloceneEIP1559Params_UsesCanyonDenominator(t *testing.T) {
	rollupCfg := map[string]any{
		"holocene_time": float64(0),
		"chain_op_config": map[string]any{
			"eip1559Elasticity":        float64(6),
			"eip1559Denominator":       float64(50),
			"eip1559DenominatorCanyon": float64(250),
		},
		"genesis": map[string]any{
			"system_config": map[string]any{
				"eip1559Params": "0x0000000000000000",
			},
		},
	}

	if err := normalizeHoloceneEIP1559Params(rollupCfg); err != nil {
		t.Fatalf("normalizeHoloceneEIP1559Params returned error: %v", err)
	}

	genesis := rollupCfg["genesis"].(map[string]any)
	systemConfig := genesis["system_config"].(map[string]any)
	got := systemConfig["eip1559Params"]
	want := "0x000000fa00000006"
	if got != want {
		t.Fatalf("unexpected eip1559Params: got %v want %v", got, want)
	}
}

func TestNormalizeHoloceneEIP1559Params_FallsBackToLegacyDenominator(t *testing.T) {
	rollupCfg := map[string]any{
		"holocene_time": float64(0),
		"chain_op_config": map[string]any{
			"eip1559Elasticity":  float64(6),
			"eip1559Denominator": float64(50),
		},
		"genesis": map[string]any{
			"system_config": map[string]any{
				"eip1559Params": "0x0000000000000000",
			},
		},
	}

	if err := normalizeHoloceneEIP1559Params(rollupCfg); err != nil {
		t.Fatalf("normalizeHoloceneEIP1559Params returned error: %v", err)
	}

	genesis := rollupCfg["genesis"].(map[string]any)
	systemConfig := genesis["system_config"].(map[string]any)
	got := systemConfig["eip1559Params"]
	want := "0x0000003200000006"
	if got != want {
		t.Fatalf("unexpected eip1559Params fallback value: got %v want %v", got, want)
	}
}
