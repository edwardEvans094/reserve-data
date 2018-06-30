package exchange

import (
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/KyberNetwork/reserve-data/common"
	"github.com/KyberNetwork/reserve-data/settings"
	ethereum "github.com/ethereum/go-ethereum/common"
)

type StableEx struct {
	setting Setting
}

func (self *StableEx) TokenAddresses() (map[string]ethereum.Address, error) {
	// returning admin multisig. In case anyone sent dgx to this address,
	// we can still get it.
	return map[string]ethereum.Address{
		"DGX": ethereum.HexToAddress("0xFDF28Bf25779ED4cA74e958d54653260af604C20"),
	}, nil
}

func (self *StableEx) MarshalText() (text []byte, err error) {
	return []byte(self.ID()), nil
}

func (self *StableEx) Address(token common.Token) (ethereum.Address, bool) {
	addrs, err := self.TokenAddresses()
	if err != nil {
		return ethereum.Address{}, false
	}
	addr, supported := addrs[token.ID]
	return addr, supported
}

func (self *StableEx) UpdateAllDepositAddresses(address string) {
	panic("dgx doesn't support update deposit addresses")
}

func (self *StableEx) UpdateDepositAddress(token common.Token, address string) error {
	panic("dgx doesn't support update deposit addresses")
}

func (self *StableEx) GetInfo() (common.ExchangeInfo, error) {
	return self.setting.GetExchangeInfo(settings.StableExchange)
}

func (self *StableEx) GetLiveExchangeInfos(tokenPairIDs []common.TokenPairID) (common.ExchangeInfo, error) {
	return common.ExchangeInfo{}, errors.New("Stable exchange doesn't support live token")
}

func (self *StableEx) GetExchangeInfo(pair common.TokenPairID) (common.ExchangePrecisionLimit, error) {
	exInfo, err := self.setting.GetExchangeInfo(settings.StableExchange)
	if err != nil {
		return common.ExchangePrecisionLimit{}, err
	}
	data, err := exInfo.Get(pair)
	return data, err
}

func (self *StableEx) GetFee() (common.ExchangeFees, error) {
	return self.setting.GetFee(settings.StableExchange)
}

// ID must return the exact string or else simulation will fail
func (self *StableEx) ID() common.ExchangeID {
	return common.ExchangeID(settings.StableExchange.String())
}

func (self *StableEx) TokenPairs() ([]common.TokenPair, error) {
	result := []common.TokenPair{}
	exInfo, err := self.setting.GetExchangeInfo(settings.StableExchange)
	if err != nil {
		return nil, err
	}
	for pair := range exInfo.GetData() {
		pairIDs := strings.Split(string(pair), "-")
		if len(pairIDs) != 2 {
			return result, fmt.Errorf("PairID %s is malformed", string(pair))
		}
		tok1, uErr := self.setting.GetTokenByID(pairIDs[0])
		if uErr != nil {
			return result, fmt.Errorf("cant get Token %s, %s", pairIDs[0], uErr)
		}
		tok2, uErr := self.setting.GetTokenByID(pairIDs[1])
		if uErr != nil {
			return result, fmt.Errorf("cant get Token %s, %s", pairIDs[1], uErr)
		}
		tokPair := common.TokenPair{
			Base:  tok1,
			Quote: tok2,
		}
		result = append(result, tokPair)
	}
	return result, nil
}

func (self *StableEx) Name() string {
	return "stable token exchange"
}

func (self *StableEx) QueryOrder(symbol string, id uint64) (done float64, remaining float64, finished bool, err error) {
	// TODO: see if trade order (a tx to dgx contract) is successful or not
	// - successful: done = order amount, remaining = 0, finished = true, err = nil
	// - failed: done = 0, remaining = order amount, finished = false, err = some error
	// - pending: done = 0, remaining = order amount, finished = false, err = nil
	return 0, 0, false, errors.New("not supported")
}

func (self *StableEx) Trade(tradeType string, base common.Token, quote common.Token, rate float64, amount float64, timepoint uint64) (id string, done float64, remaining float64, finished bool, err error) {
	// TODO: communicate with dgx connector to do the trade
	return "not supported", 0, 0, false, errors.New("not supported")
}

func (self *StableEx) Withdraw(token common.Token, amount *big.Int, address ethereum.Address, timepoint uint64) (string, error) {
	// TODO: communicate with dgx connector to withdraw
	return "not supported", errors.New("not supported")
}

func (self *StableEx) CancelOrder(id, base, quote string) error {
	return errors.New("Dgx doesn't support trade cancelling")
}

func (self *StableEx) FetchPriceData(timepoint uint64) (map[common.TokenPairID]common.ExchangePrice, error) {
	result := map[common.TokenPairID]common.ExchangePrice{}
	// TODO: Get price data from dgx connector and construct valid orderbooks
	return result, nil
}

func (self *StableEx) FetchEBalanceData(timepoint uint64) (common.EBalanceEntry, error) {
	result := common.EBalanceEntry{}
	result.Timestamp = common.Timestamp(fmt.Sprintf("%d", timepoint))
	result.Valid = true
	result.Status = true
	// TODO: Get balance data from dgx connector
	result.ReturnTime = common.GetTimestamp()
	result.AvailableBalance = map[string]float64{"DGX": 0, "ETH": 0}
	result.LockedBalance = map[string]float64{"DGX": 0, "ETH": 0}
	result.DepositBalance = map[string]float64{"DGX": 0, "ETH": 0}
	return result, nil
}

func (self *StableEx) GetTradeHistory(fromTime, toTime uint64) (common.ExchangeTradeHistory, error) {
	return common.ExchangeTradeHistory{}, nil
}

func (self *StableEx) FetchTradeHistory(timepoint uint64) (map[common.TokenPairID][]common.TradeHistory, error) {
	result := map[common.TokenPairID][]common.TradeHistory{}
	// TODO: get trade history
	return result, errors.New("not supported")
}

func (self *StableEx) DepositStatus(id common.ActivityID, txHash, currency string, amount float64, timepoint uint64) (string, error) {
	// TODO: checking txHash status
	return "", errors.New("not supported")
}

func (self *StableEx) WithdrawStatus(id, currency string, amount float64, timepoint uint64) (string, string, error) {
	// TODO: checking id (id is the txhash) status
	return "", "", errors.New("not supported")
}

func (self *StableEx) OrderStatus(id string, base, quote string) (string, error) {
	// TODO: checking id (id is the txhash) status
	return "", errors.New("not supported")
}

func (self *StableEx) GetMinDeposit() (common.ExchangesMinDeposit, error) {
	return self.setting.GetMinDeposit(settings.StableExchange)
}

func NewStableEx(setting Setting) (*StableEx, error) {
	return &StableEx{
		setting,
	}, nil
}
