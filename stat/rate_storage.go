package stat

import (
	"github.com/KyberNetwork/reserve-data/common"
	ethereum "github.com/ethereum/go-ethereum/common"
)

// RateStorage is the storage interface of conversion rates.
type RateStorage interface {
	StoreReserveRates(reserveAddr ethereum.Address, rate common.ReserveRates, timepoint uint64) error
	GetReserveRates(fromTime, toTime uint64, reserveAddr ethereum.Address) ([]common.ReserveRates, error)
}
