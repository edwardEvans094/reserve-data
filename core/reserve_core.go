package core

import (
	"errors"
	"fmt"
	"log"
	"math/big"
	"strconv"
	"time"

	"github.com/KyberNetwork/reserve-data/common"
	ethereum "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

const (
	// HIGH_BOUND_GAS_PRICE is the price we will try to use to get higher priority
	// than trade tx to avoid price front running from users.
	HIGH_BOUND_GAS_PRICE float64 = 50.1

	statusFailed    = "failed"
	statusSubmitted = "submitted"
	statusDone      = "done"
)

type ReserveCore struct {
	blockchain      Blockchain
	activityStorage ActivityStorage
	rm              ethereum.Address
}

func NewReserveCore(
	blockchain Blockchain,
	storage ActivityStorage,
	rm ethereum.Address) *ReserveCore {
	return &ReserveCore{
		blockchain,
		storage,
		rm,
	}
}

func timebasedID(id string) common.ActivityID {
	return common.NewActivityID(uint64(time.Now().UnixNano()), id)
}

func (self ReserveCore) CancelOrder(id common.ActivityID, exchange common.Exchange) error {
	activity, err := self.activityStorage.GetActivity(id)
	if err != nil {
		return err
	}
	if activity.Action != "trade" {
		return errors.New("This is not an order activity so cannot cancel")
	}
	base := activity.Params["base"].(string)
	quote := activity.Params["quote"].(string)
	orderId := id.EID
	return exchange.CancelOrder(orderId, base, quote)
}

func (self ReserveCore) GetAddresses() *common.Addresses {
	return self.blockchain.GetAddresses()
}

func (self ReserveCore) Trade(
	exchange common.Exchange,
	tradeType string,
	base common.Token,
	quote common.Token,
	rate float64,
	amount float64,
	timepoint uint64) (common.ActivityID, float64, float64, bool, error) {
	var err error

	recordActivity := func(id, status string, done, remaining float64, finished bool, err error) error {
		uid := timebasedID(id)
		log.Printf(
			"Core ----------> %s on %s: base: %s, quote: %s, rate: %s, amount: %s, timestamp: %d ==> Result: id: %s, done: %s, remaining: %s, finished: %t, error: %s",
			tradeType, exchange.ID(), base.ID, quote.ID,
			strconv.FormatFloat(rate, 'f', -1, 64),
			strconv.FormatFloat(amount, 'f', -1, 64), timepoint,
			uid,
			strconv.FormatFloat(done, 'f', -1, 64),
			strconv.FormatFloat(remaining, 'f', -1, 64),
			finished, err,
		)

		return self.activityStorage.Record(
			"trade",
			uid,
			string(exchange.ID()),
			map[string]interface{}{
				"exchange":  exchange,
				"type":      tradeType,
				"base":      base,
				"quote":     quote,
				"rate":      rate,
				"amount":    strconv.FormatFloat(amount, 'f', -1, 64),
				"timepoint": timepoint,
			}, map[string]interface{}{
				"id":        id,
				"done":      done,
				"remaining": remaining,
				"finished":  finished,
				"error":     common.ErrorToString(err),
			},
			status,
			"",
			timepoint,
		)
	}

	if err = sanityCheckTrading(exchange, base, quote, rate, amount); err != nil {
		if sErr := recordActivity("", statusFailed, 0, 0, false, err); sErr != nil {
			log.Printf("failed to save activity record: %s", sErr)
		}
		return common.ActivityID{}, 0, 0, false, err
	}

	id, done, remaining, finished, err := exchange.Trade(tradeType, base, quote, rate, amount, timepoint)
	uid := timebasedID(id)
	if err != nil {
		if sErr := recordActivity(id, statusFailed, done, remaining, finished, err); sErr != nil {
			log.Printf("failed to save activity record: %s", sErr)
		}
		return uid, done, remaining, finished, err
	}

	var status string
	if finished {
		status = statusDone
	} else {
		status = statusSubmitted
	}

	err = recordActivity(id, status, done, remaining, finished, nil)
	return uid, done, remaining, finished, err
}

func (self ReserveCore) Deposit(
	exchange common.Exchange,
	token common.Token,
	amount *big.Int,
	timepoint uint64) (common.ActivityID, error) {
	address, supported := exchange.Address(token)
	var (
		err         error
		ok          bool
		tx          *types.Transaction
		amountFloat = common.BigToFloat(amount, token.Decimal)
	)

	uidGenerator := func(txhex string) common.ActivityID {
		return timebasedID(txhex + "|" + token.ID + "|" + strconv.FormatFloat(amountFloat, 'f', -1, 64))
	}
	recordActivity := func(status, txhex, txnonce, txprice string, err error) error {
		uid := uidGenerator(txhex)
		log.Printf(
			"Core ----------> Deposit to %s: token: %s, amount: %s, timestamp: %d ==> Result: tx: %s, error: %s",
			exchange.ID(), token.ID, amount.Text(10), timepoint, txhex, err,
		)
		return self.activityStorage.Record(
			"deposit",
			uid,
			string(exchange.ID()),
			map[string]interface{}{
				"exchange":  exchange,
				"token":     token,
				"amount":    strconv.FormatFloat(amountFloat, 'f', -1, 64),
				"timepoint": timepoint,
			}, map[string]interface{}{
				"tx":       txhex,
				"nonce":    txnonce,
				"gasPrice": txprice,
				"error":    common.ErrorToString(err),
			},
			"",
			status,
			timepoint,
		)
	}

	if !supported {
		err = fmt.Errorf("Exchange %s doesn't support token %s", exchange.ID(), token.ID)
		if sErr := recordActivity(statusFailed, "", "", "", err); sErr != nil {
			log.Printf("failed to save activity record: %s", sErr)
		}
		return common.ActivityID{}, err
	}

	if ok, err = self.activityStorage.HasPendingDeposit(token, exchange); err != nil {
		if sErr := recordActivity(statusFailed, "", "", "", err); sErr != nil {
			log.Printf("failed to save activity record: %s", sErr)
		}
		return common.ActivityID{}, err
	}
	if ok {
		err = fmt.Errorf("There is a pending %s deposit to %s currently, please try again", token.ID, exchange.ID())
		if sErr := recordActivity(statusFailed, "", "", "", err); sErr != nil {
			log.Printf("failed to save activity record: %s", sErr)
		}
		return common.ActivityID{}, err
	}

	if err = sanityCheckAmount(exchange, token, amount); err != nil {
		if sErr := recordActivity(statusFailed, "", "", "", err); sErr != nil {
			log.Printf("failed to save activity record: %s", sErr)
		}
		return common.ActivityID{}, err
	}
	if tx, err = self.blockchain.Send(token, amount, address); err != nil {
		if sErr := recordActivity(statusFailed, "", "", "", err); sErr != nil {
			log.Printf("failed to save activity record: %s", sErr)
		}
		return common.ActivityID{}, err
	}

	err = recordActivity(
		statusSubmitted,
		tx.Hash().Hex(),
		strconv.FormatUint(tx.Nonce(), 10),
		tx.GasPrice().Text(10),
		nil,
	)
	return uidGenerator(tx.Hash().Hex()), err
}

func (self ReserveCore) Withdraw(
	exchange common.Exchange, token common.Token,
	amount *big.Int, timepoint uint64) (common.ActivityID, error) {
	var err error

	activityRecord := func(id, status string, err error) error {
		uid := timebasedID(id)
		log.Printf(
			"Core ----------> Withdraw from %s: token: %s, amount: %s, timestamp: %d ==> Result: id: %s, error: %s",
			exchange.ID(), token.ID, amount.Text(10), timepoint, id, err,
		)
		return self.activityStorage.Record(
			"withdraw",
			uid,
			string(exchange.ID()),
			map[string]interface{}{
				"exchange":  exchange,
				"token":     token,
				"amount":    strconv.FormatFloat(common.BigToFloat(amount, token.Decimal), 'f', -1, 64),
				"timepoint": timepoint,
			}, map[string]interface{}{
				"error": common.ErrorToString(err),
				"id":    id,
				// this field will be updated with real tx when data fetcher can fetch it
				// from exchanges
				"tx": "",
			},
			status,
			"",
			timepoint,
		)
	}

	_, supported := exchange.Address(token)
	if !supported {
		err = fmt.Errorf("Exchange %s doesn't support token %s", exchange.ID(), token.ID)
		if sErr := activityRecord("", statusFailed, err); sErr != nil {
			log.Printf("failed to store activiry record: %s", sErr.Error())
		}
		return common.ActivityID{}, err

	}

	if err = sanityCheckAmount(exchange, token, amount); err != nil {
		if sErr := activityRecord("", statusFailed, err); sErr != nil {
			log.Printf("failed to store activiry record: %s", sErr.Error())
		}
		return common.ActivityID{}, err
	}

	id, err := exchange.Withdraw(token, amount, self.rm, timepoint)
	if err != nil {
		if sErr := activityRecord("", statusFailed, err); sErr != nil {
			log.Printf("failed to store activiry record: %s", sErr.Error())
		}
		return common.ActivityID{}, err
	}

	err = activityRecord(id, statusSubmitted, nil)
	return timebasedID(id), err
}

func calculateNewGasPrice(old *big.Int, count uint64) *big.Int {
	// in this case after 5 tries the tx is still not mined.
	// at this point, 50.1 gwei is not enough but it doesn't matter
	// if the tx is mined or not because users' tx is not mined neither
	// so we can just increase the gas price a tiny amount (1 gwei) to make
	// the node accept tx with up to date price
	if count > 4 {
		return old.Add(old, common.GweiToWei(1))
	} else {
		// new = old + (50.1 - old) / (5 - count)
		return old.Add(
			old,
			big.NewInt(0).Div(big.NewInt(0).Sub(common.GweiToWei(HIGH_BOUND_GAS_PRICE), old), big.NewInt(int64(5-count))),
		)
	}
}

func (self ReserveCore) pendingSetrateInfo(minedNonce uint64) (*big.Int, *big.Int, uint64, error) {
	act, count, err := self.activityStorage.PendingSetrate(minedNonce)
	if err != nil {
		return nil, nil, 0, err
	}
	if act == nil {
		return nil, nil, 0, nil
	}
	nonce, _ := strconv.ParseUint(act.Result["nonce"].(string), 10, 64)
	gasPrice, _ := strconv.ParseUint(act.Result["gasPrice"].(string), 10, 64)
	return big.NewInt(int64(nonce)), big.NewInt(int64(gasPrice)), count, nil
}

func (self ReserveCore) SetRates(
	tokens []common.Token,
	buys []*big.Int,
	sells []*big.Int,
	block *big.Int,
	afpMids []*big.Int,
	additionalMsgs []string) (common.ActivityID, error) {

	lentokens := len(tokens)
	lenbuys := len(buys)
	lensells := len(sells)
	lenafps := len(afpMids)

	var tx *types.Transaction
	var txhex string = ethereum.Hash{}.Hex()
	var txnonce string = "0"
	var txprice string = "0"
	var err error
	var status string

	if lentokens != lenbuys || lentokens != lensells || lentokens != lenafps {
		err = errors.New("Tokens, buys sells and afpMids must have the same length")
	} else {
		err = sanityCheck(buys, afpMids, sells)
		if err == nil {
			tokenAddrs := []ethereum.Address{}
			for _, token := range tokens {
				tokenAddrs = append(tokenAddrs, ethereum.HexToAddress(token.Address))
			}
			// if there is a pending set rate tx, we replace it
			var oldNonce *big.Int
			var oldPrice *big.Int
			var minedNonce uint64
			var count uint64
			minedNonce, err = self.blockchain.SetRateMinedNonce()
			if err != nil {
				err = errors.New("Couldn't get mined nonce of set rate operator")
			} else {
				oldNonce, oldPrice, count, err = self.pendingSetrateInfo(minedNonce)
				log.Printf("old nonce: %v, old price: %v, count: %d, err: %v", oldNonce, oldPrice, count, err)
				if err != nil {
					err = errors.New("Couldn't check pending set rate tx pool. Please try later")
				} else {
					if oldNonce != nil {
						newPrice := calculateNewGasPrice(oldPrice, count)
						log.Printf("Trying to replace old tx with new price: %s", newPrice.Text(10))
						tx, err = self.blockchain.SetRates(
							tokenAddrs, buys, sells, block,
							oldNonce,
							newPrice,
						)
					} else {
						recommendedPrice := self.blockchain.StandardGasPrice()
						var initPrice *big.Int
						if recommendedPrice == 0 || recommendedPrice > HIGH_BOUND_GAS_PRICE {
							initPrice = common.GweiToWei(10)
						} else {
							initPrice = common.GweiToWei(recommendedPrice)
						}
						tx, err = self.blockchain.SetRates(
							tokenAddrs, buys, sells, block,
							big.NewInt(int64(minedNonce)),
							initPrice,
						)
					}
				}
			}
		}
	}
	if err != nil {
		status = "failed"
	} else {
		status = "submitted"
		txhex = tx.Hash().Hex()
		txnonce = strconv.FormatUint(tx.Nonce(), 10)
		txprice = tx.GasPrice().Text(10)
	}
	uid := timebasedID(txhex)
	err = self.activityStorage.Record(
		"set_rates",
		uid,
		"blockchain",
		map[string]interface{}{
			"tokens": tokens,
			"buys":   buys,
			"sells":  sells,
			"block":  block,
			"afpMid": afpMids,
			"msgs":   additionalMsgs,
		}, map[string]interface{}{
			"tx":       txhex,
			"nonce":    txnonce,
			"gasPrice": txprice,
			"error":    common.ErrorToString(err),
		},
		"",
		status,
		common.GetTimepoint(),
	)
	log.Printf(
		"Core ----------> Set rates: ==> Result: tx: %s, nonce: %s, price: %s, error: %s",
		txhex, txnonce, txprice, err,
	)
	return uid, err
}

func sanityCheck(buys, afpMid, sells []*big.Int) error {
	eth := big.NewFloat(0).SetInt(common.EthToWei(1))
	for i, s := range sells {
		check := checkZeroValue(buys[i], s)
		switch check {
		case 1:
			sFloat := big.NewFloat(0).SetInt(s)
			sRate := calculateRate(sFloat, eth)
			bFloat := big.NewFloat(0).SetInt(buys[i])
			bRate := calculateRate(eth, bFloat)
			aMFloat := big.NewFloat(0).SetInt(afpMid[i])
			aMRate := calculateRate(aMFloat, eth)
			if bRate.Cmp(sRate) <= 0 || bRate.Cmp(aMRate) <= 0 {
				return errors.New("Sell price must be bigger than buy price and afpMid price")
			}
		case 0:
			return nil
		case -1:
			return errors.New("Rate cannot be zero on only sell or buy side")
		}
	}
	return nil
}

func sanityCheckTrading(exchange common.Exchange, base, quote common.Token, rate, amount float64) error {
	tokenPairID := makeTokenPair(base.ID, quote.ID)
	exchangeInfo, err := exchange.GetExchangeInfo(tokenPairID)
	if err != nil {
		return err
	}
	currentNotional := rate * amount
	minNotional := exchangeInfo.MinNotional
	if minNotional != float64(0) {
		if currentNotional < minNotional {
			return errors.New("Notional must be bigger than exchange's MinNotional")
		}
	}
	return nil
}

func sanityCheckAmount(exchange common.Exchange, token common.Token, amount *big.Int) error {
	exchangeFee := exchange.GetFee()
	amountFloat := big.NewFloat(0).SetInt(amount)
	feeWithdrawing := exchangeFee.Funding.GetTokenFee(string(token.ID))
	expDecimal := big.NewInt(0).Exp(big.NewInt(10), big.NewInt(token.Decimal), nil)
	minAmountWithdraw := big.NewFloat(0)

	minAmountWithdraw.Mul(big.NewFloat(feeWithdrawing), big.NewFloat(0).SetInt(expDecimal))
	if amountFloat.Cmp(minAmountWithdraw) < 0 {
		return errors.New("Amount is too small!!!")
	}
	return nil
}

func calculateRate(theDividend, divisor *big.Float) *big.Float {
	div := big.NewFloat(0)
	div.Quo(theDividend, divisor)
	return div
}

func checkZeroValue(buy, sell *big.Int) int {
	zero := big.NewInt(0)
	if buy.Cmp(zero) == 0 && sell.Cmp(zero) == 0 {
		return 0
	}
	if buy.Cmp(zero) > 0 && sell.Cmp(zero) > 0 {
		return 1
	}
	return -1
}

func makeTokenPair(base, quote string) common.TokenPairID {
	if base == "ETH" {
		return common.NewTokenPairID(quote, base)
	}
	return common.NewTokenPairID(base, quote)
}
