package common

import (
	"fmt"
	"math/big"

	ethereum "github.com/ethereum/go-ethereum/common"
)

// Exchange represents a centralized exchange like Binance, Huobi...
type Exchange interface {
	ID() ExchangeID
	// Address return the deposit address of a token and return true if token is supported in the exchange.
	// Otherwise return false. This function will prioritize live address from exchange above the current stored address.
	Address(token Token) (address ethereum.Address, supported bool)
	UpdateDepositAddress(token Token, addr string) error
	Withdraw(token Token, amount *big.Int, address ethereum.Address, timepoint uint64) (string, error)
	Trade(tradeType string, base, quote Token, rate, amount float64, timepoint uint64) (id string, done, remaining float64, finished bool, err error)
	CancelOrder(id, base, quote string) error
	MarshalText() (text []byte, err error)
	GetInfo() (ExchangeInfo, error)
	GetExchangeInfo(TokenPairID) (ExchangePrecisionLimit, error)
	GetFee() (ExchangeFees, error)
	GetMinDeposit() (ExchangesMinDeposit, error)
	TokenAddresses() (map[string]ethereum.Address, error)
	GetTradeHistory(fromTime, toTime uint64) (ExchangeTradeHistory, error)
}

var SupportedExchanges = map[ExchangeID]Exchange{}

func GetExchange(id string) (Exchange, error) {
	ex := SupportedExchanges[ExchangeID(id)]
	if ex == nil {
		return ex, fmt.Errorf("Exchange %s is not supported", id)
	} else {
		return ex, nil
	}
}

func MustGetExchange(id string) Exchange {
	result, err := GetExchange(id)
	if err != nil {
		panic(err)
	}
	return result
}

type CompositeExchangeSetting struct {
	ExDepositAddress ExchangeAddresses
	ExMinDeposit     ExchangesMinDeposit
	ExFee            ExchangeFees
	ExInfo           ExchangeInfo
}

func NewCompositeExchangeSetting(depoAddr ExchangeAddresses, minDep ExchangesMinDeposit, fee ExchangeFees, info ExchangeInfo) *CompositeExchangeSetting {
	return &CompositeExchangeSetting{
		ExDepositAddress: depoAddr,
		ExMinDeposit:     minDep,
		ExFee:            fee,
		ExInfo:           info,
	}
}
