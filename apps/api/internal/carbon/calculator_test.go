package carbon

import (
	"math"
	"testing"
)

func TestCalculateCO2e_KnownValue(t *testing.T) {
	// phi_3, 80 ms, batch=1 → energyKwh = 3640/3_600_000_000 ≈ 1.01111e-6
	// US-EAST intensity = 450 gCO2/kWh
	// co2e = 1.01111e-6 × 450 × 1000 = 0.455 g
	//      = 3640 × 450 / 3_600_000 = 1_638_000 / 3_600_000 = 0.455
	const (
		energyKwh = 3640.0 / 3_600_000_000.0
		wantCO2e  = 1_638_000.0 / 3_600_000.0 // ≈ 0.455 g
	)

	grid := GridData{Region: "US-EAST", IntensityGCO2: 450, DataSource: "static"}
	est := CalculateCO2e(energyKwh, grid)

	if math.Abs(est.CO2eGrams-wantCO2e) > 1e-9 {
		t.Errorf("CO2eGrams = %.6f, want %.6f", est.CO2eGrams, wantCO2e)
	}
	if est.GridRegion != "US-EAST" {
		t.Errorf("GridRegion = %q, want US-EAST", est.GridRegion)
	}
	if est.GridCarbonIntensity != 450 {
		t.Errorf("GridCarbonIntensity = %g, want 450", est.GridCarbonIntensity)
	}
	if est.GPT4EquivalentCO2e != GPT4BaselineCO2eGrams {
		t.Errorf("GPT4EquivalentCO2e = %g, want %g", est.GPT4EquivalentCO2e, GPT4BaselineCO2eGrams)
	}
}

func TestCalculateCO2e_LowCarbonGrid_LessThanTenPercentHighCarbon(t *testing.T) {
	// EU-NO (40 gCO2/kWh) vs US-EAST (450 gCO2/kWh): ratio must be < 10%.
	energyKwh := 1e-6
	highCarbon := CalculateCO2e(energyKwh, GridData{Region: "US-EAST", IntensityGCO2: 450})
	lowCarbon := CalculateCO2e(energyKwh, GridData{Region: "EU-NO", IntensityGCO2: 40})

	if highCarbon.CO2eGrams == 0 {
		t.Fatal("high-carbon estimate should not be zero")
	}
	ratio := lowCarbon.CO2eGrams / highCarbon.CO2eGrams
	if ratio >= 0.1 {
		t.Errorf("EU-NO/US-EAST CO2e ratio = %.4f, want < 0.10 (40/450 ≈ 0.089)", ratio)
	}
}

func TestCalculateCO2e_GPT4Savings_LargeForSmallModel(t *testing.T) {
	// phi_3 at 80 ms on US-EAST → CO2e ≈ 0.455 g vs GPT-4 36 g → >80% savings.
	energyKwh := 3640.0 / 3_600_000_000.0
	est := CalculateCO2e(energyKwh, GridData{Region: "US-EAST", IntensityGCO2: 450, DataSource: "static"})

	if est.SavingsPercent < 80 {
		t.Errorf("SavingsPercent = %.1f%%, want > 80%% for phi_3 vs GPT-4", est.SavingsPercent)
	}
	if est.SavingsPercent > 100 {
		t.Errorf("SavingsPercent = %.1f%%, must not exceed 100%%", est.SavingsPercent)
	}
}

func TestCalculateGPT4Comparison_HighEnergyZeroSavings(t *testing.T) {
	// A model that equals or exceeds GPT-4 baseline should report 0% savings.
	_, savings := CalculateGPT4Comparison(GPT4BaselineCO2eGrams)
	if savings != 0 {
		t.Errorf("savings = %.2f%%, want 0%% when CO2e equals GPT-4 baseline", savings)
	}

	_, savingsOver := CalculateGPT4Comparison(GPT4BaselineCO2eGrams * 2)
	if savingsOver != 0 {
		t.Errorf("savings = %.2f%%, want 0%% when CO2e exceeds GPT-4 baseline", savingsOver)
	}
}

func TestCalculateGPT4Comparison_SavingsProportional(t *testing.T) {
	// 50% of baseline CO2e → exactly 50% savings.
	gpt4, savings := CalculateGPT4Comparison(GPT4BaselineCO2eGrams / 2)

	if gpt4 != GPT4BaselineCO2eGrams {
		t.Errorf("gpt4CO2e = %g, want %g", gpt4, GPT4BaselineCO2eGrams)
	}
	if math.Abs(savings-50.0) > 1e-9 {
		t.Errorf("savings = %.6f%%, want 50%% for half-baseline CO2e", savings)
	}
}
