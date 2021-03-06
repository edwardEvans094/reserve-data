package metric

import (
	"github.com/KyberNetwork/reserve-data/common"
)

// MetricStorage is the interface that wraps all metrics database operations.
type MetricStorage interface {
	StoreMetric(data *MetricEntry, timepoint uint64) error
	StoreTokenTargetQty(id, data string) error
	StorePendingTargetQty(data, dataType string) error
	StoreRebalanceControl(status bool) error
	StoreSetrateControl(status bool) error
	StorePendingPWIEquation(data string) error
	StorePWIEquation(data string) error

	GetMetric(tokens []common.Token, fromTime, toTime uint64) (map[string]MetricList, error)
	GetTokenTargetQty() (TokenTargetQty, error)
	GetPendingTargetQty() (TokenTargetQty, error)
	GetRebalanceControl() (RebalanceControl, error)
	GetSetrateControl() (SetrateControl, error)
	GetPendingPWIEquation() (PWIEquation, error)
	GetPWIEquation() (PWIEquation, error)

	RemovePendingTargetQty() error
	RemovePendingPWIEquation() error

	SetStableTokenParams(value []byte) error
	ConfirmStableTokenParams(value []byte) error
	RemovePendingStableTokenParams() error
	GetPendingStableTokenParams() (map[string]interface{}, error)
	GetStableTokenParams() (map[string]interface{}, error)

	StorePendingTargetQtyV2(value []byte) error
	ConfirmTargetQtyV2(value []byte) error
	RemovePendingTargetQtyV2() error
	GetPendingTargetQtyV2() (TokenTargetQtyV2, error)
	GetTargetQtyV2() (TokenTargetQtyV2, error)

	StorePendingPWIEquationV2([]byte) error
	GetPendingPWIEquationV2() (PWIEquationRequestV2, error)
	StorePWIEquationV2(data string) error
	RemovePendingPWIEquationV2() error
	GetPWIEquationV2() (PWIEquationRequestV2, error)

	StorePendingRebalanceQuadratic([]byte) error
	GetPendingRebalanceQuadratic() (RebalanceQuadraticRequest, error)
	ConfirmRebalanceQuadratic(data []byte) error
	RemovePendingRebalanceQuadratic() error
	GetRebalanceQuadratic() (RebalanceQuadraticRequest, error)
}
