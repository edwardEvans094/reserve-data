package http

import (
	"github.com/KyberNetwork/reserve-data/common"
	"github.com/KyberNetwork/reserve-data/settings"
	ethereum "github.com/ethereum/go-ethereum/common"
)

type Setting interface {
	GetInternalTokenByID(tokenID string) (common.Token, error)
	GetActiveTokenByID(tokenID string) (common.Token, error)
	GetTokenByID(tokenID string) (common.Token, error)
	GetInternalTokens() ([]common.Token, error)
	GetAllTokens() ([]common.Token, error)
	NewTokenPairFromID(base, quote string) (common.TokenPair, error)
	UpdateToken(t common.Token) error
	AddAddressToSet(setName settings.AddressSetName, address ethereum.Address) error
	UpdateAddress(name settings.AddressName, address ethereum.Address) error
	GetFee(ex settings.ExchangeName) (common.ExchangeFees, error)
	UpdateFee(ex settings.ExchangeName, data common.ExchangeFees) error
	GetMinDeposit(ex settings.ExchangeName) (common.ExchangesMinDeposit, error)
	UpdateMinDeposit(ex settings.ExchangeName, minDeposit common.ExchangesMinDeposit) error
	GetDepositAddresses(ex settings.ExchangeName) (common.ExchangeAddresses, error)
	UpdateDepositAddress(ex settings.ExchangeName, addrs common.ExchangeAddresses) error
	GetExchangeInfo(ex settings.ExchangeName) (common.ExchangeInfo, error)
	UpdateExchangeInfo(ex settings.ExchangeName, exInfo common.ExchangeInfo) error
	GetExchangeStatus() (common.ExchangesStatus, error)
	UpdateExchangeStatus(data common.ExchangesStatus) error
	UpdatePendingTokenListings(map[string]common.TokenListing) error
	ApplyTokenWithExchangeSetting([]common.Token, map[settings.ExchangeName]*common.CompositeExchangeSetting) error
	GetPendingTokenListings() (map[string]common.TokenListing, error)
	RemovePendingTokenListings() error
}
