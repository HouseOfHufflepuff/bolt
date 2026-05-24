package domain

const (
	SymbolUSDL = "USDL"
	SymbolCHFL = "CHFL"
	SymbolNOKL = "NOKL"
	SymbolBOLT = "BOLT"
	SymbolBTC  = "BTC"
)

// Weights holds target basket allocation percentages. Must sum to 1.0.
type Weights struct {
	USDL float64
	CHFL float64
	NOKL float64
	BTC  float64
}

var DefaultWeights = Weights{
	USDL: 0.50,
	CHFL: 0.30,
	NOKL: 0.10,
	BTC:  0.10,
}

func (w Weights) AsMap() map[string]float64 {
	return map[string]float64{
		SymbolUSDL: w.USDL,
		SymbolCHFL: w.CHFL,
		SymbolNOKL: w.NOKL,
		SymbolBTC:  w.BTC,
	}
}

type BasketNAV struct {
	TotalUSD    float64
	Breakdown   map[string]float64 // asset → USD contribution
	WeightDrift map[string]float64 // asset → (effective weight − target weight)
}

type MintRequest struct {
	AssetSymbol string
	Amount      int64 // base units
	Operator    string
}

type MintResult struct {
	BatchKey   []byte
	AssetProof []byte
}

type BurnRequest struct {
	AssetSymbol string
	Amount      int64
	Operator    string
}

type ReserveSnapshot struct {
	AssetSymbol   string
	ReserveAmount int64
	SupplyAmount  int64
	Ratio         float64
}
