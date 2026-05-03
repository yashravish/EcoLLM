package carbon

const (
	// gpt4EnergyKwh is the published GPT-4 energy baseline per request.
	// Source: AGENT_KNOWLEDGE_BASE Layer 6 / Architecture Appendix B.
	gpt4EnergyKwh = 0.00008

	// gpt4GridIntensity is the US average grid intensity used for the GPT-4 CO2 baseline.
	gpt4GridIntensity = 450.0 // gCO2/kWh

	// GPT4BaselineCO2eGrams is the reference CO2e per GPT-4 request for comparison reporting.
	// = 0.00008 kWh × 450 gCO2/kWh × 1000 = 36 g
	GPT4BaselineCO2eGrams = gpt4EnergyKwh * gpt4GridIntensity * 1000
)

// CalculateCO2e derives a CarbonEstimate from energy and live grid data.
// It also computes the GPT-4 comparison fields for transparency reporting.
func CalculateCO2e(energyKwh float64, grid GridData) CarbonEstimate {
	co2eGrams := energyKwh * grid.IntensityGCO2 * 1000
	_, savings := CalculateGPT4Comparison(co2eGrams)

	return CarbonEstimate{
		GridRegion:          grid.Region,
		GridCarbonIntensity: grid.IntensityGCO2,
		CarbonDataSource:    grid.DataSource,
		CO2eGrams:           co2eGrams,
		GPT4EquivalentCO2e:  GPT4BaselineCO2eGrams,
		SavingsPercent:      savings,
	}
}

// CalculateGPT4Comparison returns the GPT-4 CO2e baseline and the percentage
// savings achieved by the EcoLLM request relative to that baseline.
// Returns 0% savings when the EcoLLM request meets or exceeds the GPT-4 baseline.
func CalculateGPT4Comparison(co2eGrams float64) (gpt4CO2eGrams, savingsPercent float64) {
	if co2eGrams >= GPT4BaselineCO2eGrams {
		return GPT4BaselineCO2eGrams, 0
	}
	savings := (GPT4BaselineCO2eGrams - co2eGrams) / GPT4BaselineCO2eGrams * 100
	return GPT4BaselineCO2eGrams, savings
}
