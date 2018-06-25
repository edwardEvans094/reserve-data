package metric

type TokenMetric struct {
	AfpMid float64
	Spread float64
}

type MetricEntry struct {
	Timestamp uint64
	// data contain all token metric for all tokens
	Data map[string]TokenMetric
}

type TokenMetricResponse struct {
	Timestamp uint64
	AfpMid    float64
	Spread    float64
}

// Metric list for one token
type MetricList []TokenMetricResponse

type MetricResponse struct {
	Timestamp  uint64
	ReturnTime uint64
	Data       map[string]MetricList
}

type TokenTargetQty struct {
	ID        uint64
	Timestamp uint64
	Data      string
	Status    string
	Type      int64
}

type PWIEquation struct {
	ID   uint64 `json:"id"`
	Data string `json:"data"`
}

type RebalanceControl struct {
	Status bool `json:"status"`
}

type SetrateControl struct {
	Status bool `json:"status"`
}

type TargetQtySet struct {
	TotalTarget        float64 `json:"total_target"`
	ReserveTarget      float64 `json:"reserve_target"`
	RebalanceThreshold float64 `json:"rebalance_threshold"`
	TransferThreshold  float64 `json:"transfer_threshold"`
}

//TargetQtyStruct object for save target qty
type TargetQtyStruct struct {
	SetTarget TargetQtySet `json:"set_target"`
}

// PWIEquationV2 contains the information of a PWI equation.
type PWIEquationV2 struct {
	A                   float64 `json:"a"`
	B                   float64 `json:"b"`
	C                   float64 `json:"c"`
	MinMinSpread        float64 `json:"min_min_spread"`
	PriceMultiplyFactor float64 `json:"price_multiply_factor"`
}

// PWIEquationTokenV2 is a mapping between a token id and a PWI equation.
type PWIEquationTokenV2 map[string]PWIEquationV2

// isValid validates the input instance and return true if it is valid.
// Example:
// {
//  "bid": {
//    "a": "750",
//    "b": "500",
//    "c": "0",
//    "min_min_spread": "0",
//    "price_multiply_factor": "0"
//  },
//  "ask": {
//    "a": "800",
//    "b": "600",
//    "c": "0",
//    "min_min_spread": "0",
//    "price_multiply_factor": "0"
//  }
//}
func (et PWIEquationTokenV2) isValid() bool {
	var requiredFields = []string{"bid", "ask"}
	// validates that both bid and ask are present
	if len(et) != len(requiredFields) {
		return false
	}

	for _, field := range requiredFields {
		if _, ok := et[field]; !ok {
			return false
		}
	}
	return true
}

// PWIEquationRequestV2 is the input SetPWIEquationV2 api.
type PWIEquationRequestV2 map[string]PWIEquationTokenV2

// IsValid validates the input instance and return true if it is valid.
// Example input:
// [{"token_id": {equation_token}}, ...]
func (input PWIEquationRequestV2) IsValid() bool {
	for _, et := range input {
		if !et.isValid() {
			return false
		}
	}

	return true
}

//RebalanceQuadraticEquation represent an equation
type RebalanceQuadraticEquation struct {
	RebalanceQuadratic struct {
		A float64 `json:"a"`
		B float64 `json:"b"`
		C float64 `json:"c"`
	} `json:"rebalance_quadratic"`
}

//RebalanceQuadraticRequest represent data request to set rebalance quadratic
//map[token]equation
type RebalanceQuadraticRequest map[string]RebalanceQuadraticEquation
