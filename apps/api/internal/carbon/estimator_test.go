package carbon

import (
	"math"
	"testing"
)

func TestCalculateEnergy_KnownValue_Phi3(t *testing.T) {
	// Phi-3: 35 W, 80 ms, batch=1, PUE=1.3
	// totalEnergyKwh = 35 × (80/3_600_000) × 1.3 / 1000
	//                = 3640 / 3_600_000_000
	//                ≈ 1.01111e-6 kWh
	const want = 3640.0 / 3_600_000_000.0

	m := CalculateEnergy(EnergyInput{
		ModelName:     "phi_3",
		InferenceMs:   80,
		BatchSize:     1,
		PUEMultiplier: 1.3,
	})

	if math.Abs(m.TotalEnergyKwh-want) > 1e-12 {
		t.Errorf("TotalEnergyKwh = %.8e, want %.8e", m.TotalEnergyKwh, want)
	}
	if m.GPUPowerWatts != 35 {
		t.Errorf("GPUPowerWatts = %g, want 35", m.GPUPowerWatts)
	}
	if m.PUEMultiplier != 1.3 {
		t.Errorf("PUEMultiplier = %g, want 1.3", m.PUEMultiplier)
	}
	if m.BatchSize != 1 {
		t.Errorf("BatchSize = %d, want 1", m.BatchSize)
	}
	if m.InferenceTimeMs != 80 {
		t.Errorf("InferenceTimeMs = %d, want 80", m.InferenceTimeMs)
	}
}

func TestCalculateEnergy_Llama70B_ExceedsTenTimesPhi3(t *testing.T) {
	// phi_3 at 80 ms (p95 latency), llama_70b at 800 ms (10× longer for 70B params).
	// Power ratio: 250/35 ≈ 7.1×; latency ratio: 10×; combined ≈ 71× > 10×.
	phi3 := CalculateEnergy(EnergyInput{ModelName: "phi_3", InferenceMs: 80, BatchSize: 1})
	llama := CalculateEnergy(EnergyInput{ModelName: "llama_70b", InferenceMs: 800, BatchSize: 1})

	if llama.TotalEnergyKwh <= phi3.TotalEnergyKwh*10 {
		t.Errorf("llama_70b (%.3e kWh) should be >10× phi_3 (%.3e kWh); ratio=%.1f",
			llama.TotalEnergyKwh, phi3.TotalEnergyKwh,
			llama.TotalEnergyKwh/phi3.TotalEnergyKwh)
	}
}

func TestCalculateEnergy_BatchAmortization(t *testing.T) {
	// batch=4 should consume exactly 1/4 of the energy of batch=1.
	single := CalculateEnergy(EnergyInput{ModelName: "mistral_7b", InferenceMs: 200, BatchSize: 1})
	batched := CalculateEnergy(EnergyInput{ModelName: "mistral_7b", InferenceMs: 200, BatchSize: 4})

	want := single.TotalEnergyKwh / 4
	if math.Abs(batched.TotalEnergyKwh-want) > 1e-15 {
		t.Errorf("batch=4 energy %.8e, want %.8e (single/4)", batched.TotalEnergyKwh, want)
	}
	if batched.BatchSize != 4 {
		t.Errorf("BatchSize = %d, want 4", batched.BatchSize)
	}
}

func TestCalculateEnergy_ZeroInferenceTime(t *testing.T) {
	m := CalculateEnergy(EnergyInput{ModelName: "phi_3", InferenceMs: 0, BatchSize: 1})
	if m.TotalEnergyKwh != 0 {
		t.Errorf("zero inference time should yield zero energy, got %.8e", m.TotalEnergyKwh)
	}
	if m.InferenceEnergyWh != 0 {
		t.Errorf("zero inference time should yield zero InferenceEnergyWh, got %g", m.InferenceEnergyWh)
	}
}

func TestCalculateEnergy_PUEImpact(t *testing.T) {
	// PUE=1.3 should yield exactly 1.3× the energy of PUE=1.0.
	base := CalculateEnergy(EnergyInput{ModelName: "phi_3", InferenceMs: 500, BatchSize: 1, PUEMultiplier: 1.0})
	high := CalculateEnergy(EnergyInput{ModelName: "phi_3", InferenceMs: 500, BatchSize: 1, PUEMultiplier: 1.3})

	if base.TotalEnergyKwh == 0 {
		t.Fatal("base energy should not be zero")
	}
	ratio := high.TotalEnergyKwh / base.TotalEnergyKwh
	if math.Abs(ratio-1.3) > 1e-9 {
		t.Errorf("PUE ratio = %.10f, want 1.3", ratio)
	}
}
