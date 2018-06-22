package common

import (
	ethereum "github.com/ethereum/go-ethereum/common"
)

type TokenInfo struct {
	Address  ethereum.Address `json:"address"`
	Decimals int64            `json:"decimals"`
}

type Addresses struct {
	Tokens               map[string]TokenInfo          `json:"tokens"`
	Exchanges            map[ExchangeID]TokenAddresses `json:"exchanges"`
	WrapperAddress       ethereum.Address              `json:"wrapper"`
	PricingAddress       ethereum.Address              `json:"pricing"`
	ReserveAddress       ethereum.Address              `json:"reserve"`
	FeeBurnerAddress     ethereum.Address              `json:"feeburner"`
	NetworkAddress       ethereum.Address              `json:"network"`
	PricingOperator      ethereum.Address              `json:"pricing_operator"`
	DepositOperator      ethereum.Address              `json:"deposit_opeartor"`
	IntermediateOperator ethereum.Address              `json:"intermediate_operator"`
}

func NewAddresses(
	Tokens map[string]TokenInfo,
	Exchanges map[ExchangeID]TokenAddresses,
	WrapperAddress ethereum.Address,
	PricingAddress ethereum.Address,
	ReserveAddress ethereum.Address,
	FeeBurnerAddress ethereum.Address,
	NetworkAddress ethereum.Address,
	PricingOperator ethereum.Address,
	DepositOperator ethereum.Address,
	IntermediateOperator ethereum.Address,
) *Addresses {
	return &Addresses{
		Tokens,
		Exchanges,
		WrapperAddress,
		PricingAddress,
		ReserveAddress,
		FeeBurnerAddress,
		NetworkAddress,
		PricingOperator,
		DepositOperator,
		IntermediateOperator,
	}
}

type TokenAddresses map[string]ethereum.Address
