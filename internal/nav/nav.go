package nav

import "github.com/houseofhufflepuff/bolt/internal/domain"

// GenesisUnits computes how many units of each asset back 1 BOLT at launch,
// given the basket target weights and the prices at genesis.
// e.g. if target weight for BTC is 0.10 and BTC = $65000, then
// genesisUnits["BTC"] = 0.10 / 65000 = 0.00000154 BTC per BOLT.
func GenesisUnits(targets domain.Weights, genesisPrices map[string]float64) map[string]float64 {
	units := make(map[string]float64, 4)
	for symbol, weight := range targets.AsMap() {
		if p := genesisPrices[symbol]; p > 0 {
			units[symbol] = weight / p
		}
	}
	return units
}

// Compute returns the current NAV of 1 BOLT in USD, plus per-asset breakdown
// and weight drift from targets.
//
// genesisUnits: output of GenesisUnits(), stored at platform launch.
// prices: current USD price per asset symbol (from oracle).
func Compute(genesisUnits map[string]float64, targets domain.Weights, prices map[string]float64) domain.BasketNAV {
	targetMap := targets.AsMap()
	breakdown := make(map[string]float64, len(genesisUnits))
	total := 0.0

	for symbol, units := range genesisUnits {
		contribution := units * prices[symbol]
		breakdown[symbol] = contribution
		total += contribution
	}

	drift := make(map[string]float64, len(targetMap))
	for symbol, target := range targetMap {
		effective := 0.0
		if total > 0 {
			effective = breakdown[symbol] / total
		}
		drift[symbol] = effective - target
	}

	return domain.BasketNAV{
		TotalUSD:    total,
		Breakdown:   breakdown,
		WeightDrift: drift,
	}
}
