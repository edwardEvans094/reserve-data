package exchange

import (
	"errors"
	"fmt"
	"log"
	"math/big"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/KyberNetwork/reserve-data/common"
	"github.com/KyberNetwork/reserve-data/common/blockchain"
	huobiblockchain "github.com/KyberNetwork/reserve-data/exchange/huobi/blockchain"
	huobihttp "github.com/KyberNetwork/reserve-data/exchange/huobi/http"
	"github.com/KyberNetwork/reserve-data/settings"
	ethereum "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

const (
	HUOBI_EPSILON float64 = 0.0000000001 // 10e-10
)

type Huobi struct {
	interf     HuobiInterface
	blockchain HuobiBlockchain
	storage    HuobiStorage
	setting    Setting
}

func (self *Huobi) MarshalText() (text []byte, err error) {
	return []byte(self.ID()), nil
}

func (self *Huobi) TokenAddresses() (map[string]ethereum.Address, error) {
	addrs, err := self.setting.GetDepositAddresses(settings.Huobi)
	if err != nil {
		return nil, err
	}
	return addrs.GetData(), nil
}

// RealDepositAddress return the actual Huobi deposit address of a token
// It should only be used to send 2nd transaction.
func (self *Huobi) RealDepositAddress(tokenID string) (ethereum.Address, error) {
	liveAddress, err := self.interf.GetDepositAddress(tokenID)
	if err != nil || liveAddress.Address == "" {
		log.Printf("WARNING: Get Huobi live deposit address for token %s failed: (%v) or the replied address is empty. Check the currently available address instead", tokenID, err)
		addrs, uErr := self.setting.GetDepositAddresses(settings.Huobi)
		if uErr != nil {
			return ethereum.Address{}, uErr
		}
		result, supported := addrs.Get(tokenID)
		if !supported {
			return result, fmt.Errorf("Real deposit address of token %s is not available", tokenID)
		}
		return result, nil
	}
	return ethereum.HexToAddress(liveAddress.Address), nil
}

// Address return the deposit address of a token in Huobi exchange.
// Due to the logic of Huobi exchange, every token if supported will be
// deposited to an Intermediator address instead.
func (self *Huobi) Address(token common.Token) (ethereum.Address, bool) {
	result := self.blockchain.GetIntermediatorAddr()
	_, err := self.RealDepositAddress(token.ID)
	//if the realDepositAddress can not be querried, that mean the token isn't supported on Huobi
	if err != nil {
		return result, false
	}
	return result, true
}

// UpdateDepositAddress update the deposit address of a token in Huobi
// It will prioritize the live address over the input address
func (self *Huobi) UpdateDepositAddress(token common.Token, address string) error {
	liveAddress, err := self.interf.GetDepositAddress(token.ID)
	if err != nil || liveAddress.Address == "" {
		log.Printf("WARNING: Get Huobi live deposit address for token %s failed: (%v) or the replied address is empty. Check the currently available address instead", token.ID, err)
		addrs := common.NewExchangeAddresses()
		addrs.Update(token.ID, ethereum.HexToAddress(address))
		return self.setting.UpdateDepositAddress(settings.Huobi, *addrs)
	}
	log.Printf("Got Huobi live deposit address for token %s, attempt to update it to current setting", token.ID)
	addrs := common.NewExchangeAddresses()
	addrs.Update(token.ID, ethereum.HexToAddress(liveAddress.Address))
	return self.setting.UpdateDepositAddress(settings.Huobi, *addrs)
}

// GetLiveExchangeInfo querry the Exchange Endpoint for exchange precision and limit of a certain pair ID
// It return error if occurs.
func (self *Huobi) GetLiveExchangeInfo(tokenPairID common.TokenPairID) (common.ExchangePrecisionLimit, error) {
	var result common.ExchangePrecisionLimit
	exchangeInfo, err := self.interf.GetExchangeInfo()
	if err != nil {
		return result, err
	}
	result, ok := self.getPrecisionLimitFromSymbols(tokenPairID, exchangeInfo)
	if !ok {
		return result, fmt.Errorf("Huobi Exchange Info reply doesn't contain token pair %s", string(tokenPairID))
	}
	return result, nil
}

// getPrecisionLimitFromSymbols find the pairID amongs symbols from exchanges,
// return ExchangePrecisionLimit of that pair and true if the pairID exist amongs symbols, false if otherwise
func (self *Huobi) getPrecisionLimitFromSymbols(pair common.TokenPairID, symbols HuobiExchangeInfo) (common.ExchangePrecisionLimit, bool) {
	var result common.ExchangePrecisionLimit
	pairName := strings.ToUpper(strings.Replace(string(pair), "-", "", 1))
	for _, symbol := range symbols.Data {
		symbolName := strings.ToUpper(symbol.Base + symbol.Quote)
		if symbolName == pairName {
			result.Precision.Amount = symbol.AmountPrecision
			result.Precision.Price = symbol.PricePrecision
			result.MinNotional = 0.02
			return result, true
		}
	}
	return result, false
}

func (self *Huobi) UpdatePairsPrecision() error {
	exchangeInfo, err := self.interf.GetExchangeInfo()
	if err != nil {
		return err
	}
	exInfo, err := self.GetInfo()
	if err != nil {
		return fmt.Errorf("INFO: Can't get Exchange Info for Huobi from persistent storage (%s)", err)
	}
	if exInfo == nil {
		return errors.New("Exchange info of Huobi is nil")
	}
	for pair := range exInfo.GetData() {
		epl, exist := self.getPrecisionLimitFromSymbols(pair, exchangeInfo)
		if !exist {
			return fmt.Errorf("Huobi Exchange Info reply doesn't contain token pair %s", pair)
		}
		exInfo[pair] = epl
	}
	return self.setting.UpdateExchangeInfo(settings.Huobi, exInfo)
}

func (self *Huobi) GetInfo() (common.ExchangeInfo, error) {
	return self.setting.GetExchangeInfo(settings.Huobi)
}

func (self *Huobi) GetExchangeInfo(pair common.TokenPairID) (common.ExchangePrecisionLimit, error) {
	exInfo, err := self.setting.GetExchangeInfo(settings.Huobi)
	if err != nil {
		return common.ExchangePrecisionLimit{}, err
	}
	data, err := exInfo.Get(pair)
	return data, err
}

func (self *Huobi) GetFee() (common.ExchangeFees, error) {
	return self.setting.GetFee(settings.Huobi)
}

func (self *Huobi) GetMinDeposit() (common.ExchangesMinDeposit, error) {
	return self.setting.GetMinDeposit(settings.Huobi)
}

// ID must return the exact string or else simulation will fail
func (self *Huobi) ID() common.ExchangeID {
	return common.ExchangeID(settings.Huobi.String())
}

func (self *Huobi) TokenPairs() ([]common.TokenPair, error) {
	result := []common.TokenPair{}
	exInfo, err := self.setting.GetExchangeInfo(settings.Huobi)
	if err != nil {
		return nil, err
	}
	for pair := range exInfo.GetData() {
		pairIDs := strings.Split(string(pair), "-")
		if len(pairIDs) != 2 {
			return result, fmt.Errorf("Huobi PairID %s is malformed", string(pair))
		}
		tok1, uErr := self.setting.GetTokenByID(pairIDs[0])
		if uErr != nil {
			return result, fmt.Errorf("Huobi cant get Token %s, %s", pairIDs[0], uErr)
		}
		tok2, uErr := self.setting.GetTokenByID(pairIDs[1])
		if uErr != nil {
			return result, fmt.Errorf("Huobi cant get Token %s, %s", pairIDs[1], uErr)
		}
		tokPair := common.TokenPair{
			Base:  tok1,
			Quote: tok2,
		}
		result = append(result, tokPair)
	}
	return result, nil
}

func (self *Huobi) Name() string {
	return "huobi"
}

func (self *Huobi) QueryOrder(symbol string, id uint64) (done float64, remaining float64, finished bool, err error) {
	result, err := self.interf.OrderStatus(symbol, id)
	if err != nil {
		return 0, 0, false, err
	}
	if result.Data.ExecutedQty != "" {
		done, err = strconv.ParseFloat(result.Data.ExecutedQty, 64)
		if err != nil {
			return 0, 0, false, err
		}
	}
	var total float64
	if result.Data.OrigQty != "" {
		total, err = strconv.ParseFloat(result.Data.OrigQty, 64)
		if err != nil {
			return 0, 0, false, err
		}
	}
	return done, total - done, total-done < HUOBI_EPSILON, nil
}

func (self *Huobi) Trade(tradeType string, base common.Token, quote common.Token, rate float64, amount float64, timepoint uint64) (id string, done float64, remaining float64, finished bool, err error) {
	result, err := self.interf.Trade(tradeType, base, quote, rate, amount, timepoint)

	if err != nil {
		return "", 0, 0, false, err
	}
	var orderID uint64
	if result.OrderID != "" {
		orderID, err = strconv.ParseUint(result.OrderID, 10, 64)
		if err != nil {
			return "", 0, 0, false, err
		}
	}
	done, remaining, finished, err = self.QueryOrder(
		base.ID+quote.ID,
		orderID,
	)
	if err != nil {
		log.Printf("Huobi Query order error: %s", err.Error())
	}
	return result.OrderID, done, remaining, finished, err
}

func (self *Huobi) Withdraw(token common.Token, amount *big.Int, address ethereum.Address, timepoint uint64) (string, error) {
	withdrawID, err := self.interf.Withdraw(token, amount, address)
	if err != nil {
		return "", err
	}
	// this magical logic base on inspection on huobi website
	result := withdrawID + "01"
	return result, err
}

func (self *Huobi) CancelOrder(id, base, quote string) error {
	idNo, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		return err
	}
	symbol := base + quote
	result, err := self.interf.CancelOrder(symbol, idNo)
	if err != nil {
		return err
	}
	if result.Status != "ok" {
		return errors.New("Huobi Couldn't cancel order id " + id)
	}
	return nil
}

func (self *Huobi) FetchOnePairData(
	wg *sync.WaitGroup,
	pair common.TokenPair,
	data *sync.Map,
	timepoint uint64) {

	defer wg.Done()
	result := common.ExchangePrice{}

	timestamp := common.Timestamp(fmt.Sprintf("%d", timepoint))
	result.Timestamp = timestamp
	result.Valid = true
	resp_data, err := self.interf.GetDepthOnePair(pair)
	returnTime := common.GetTimestamp()
	result.ReturnTime = returnTime
	if err != nil {
		result.Valid = false
		result.Error = err.Error()
	} else {
		if resp_data.Status != "ok" {
			result.Valid = false
		} else {
			for _, buy := range resp_data.Tick.Bids {
				quantity := buy[1]
				rate := buy[0]
				result.Bids = append(
					result.Bids,
					common.NewPriceEntry(
						quantity,
						rate,
					),
				)
			}
			for _, sell := range resp_data.Tick.Asks {
				quantity := sell[1]
				rate := sell[0]
				result.Asks = append(
					result.Asks,
					common.NewPriceEntry(
						quantity,
						rate,
					),
				)
			}
		}
	}
	data.Store(pair.PairID(), result)
}

func (self *Huobi) FetchPriceData(timepoint uint64) (map[common.TokenPairID]common.ExchangePrice, error) {
	wait := sync.WaitGroup{}
	data := sync.Map{}
	pairs, err := self.TokenPairs()
	if err != nil {
		return nil, err
	}
	for _, pair := range pairs {
		wait.Add(1)
		go self.FetchOnePairData(&wait, pair, &data, timepoint)
	}
	wait.Wait()
	result := map[common.TokenPairID]common.ExchangePrice{}
	data.Range(func(key, value interface{}) bool {
		tokenPairID, ok := key.(common.TokenPairID)
		//if there is conversion error, continue to next key,val
		if !ok {
			err = fmt.Errorf("Key (%v) cannot be asserted to TokenPairID", key)
			return false
		}
		exPrice, ok := value.(common.ExchangePrice)
		if !ok {
			err = fmt.Errorf("Value (%v) cannot be asserted to ExchangePrice", value)
			return false
		}
		result[tokenPairID] = exPrice
		return true
	})
	return result, err
}

func (self *Huobi) OpenOrdersForOnePair(
	wg *sync.WaitGroup,
	pair common.TokenPair,
	data *sync.Map,
	timepoint uint64) {

	// defer wg.Done()

	// result, err := self.interf.OpenOrdersForOnePair(pair, timepoint)

	//TODO: complete open orders for one pair
}

func (self *Huobi) FetchOrderData(timepoint uint64) (common.OrderEntry, error) {
	result := common.OrderEntry{}
	result.Timestamp = common.Timestamp(fmt.Sprintf("%d", timepoint))
	result.Valid = true
	result.Data = []common.Order{}

	wait := sync.WaitGroup{}
	data := sync.Map{}
	pairs, err := self.TokenPairs()
	if err != nil {
		return result, err
	}
	for _, pair := range pairs {
		wait.Add(1)
		go self.OpenOrdersForOnePair(&wait, pair, &data, timepoint)
	}
	wait.Wait()

	result.ReturnTime = common.GetTimestamp()
	data.Range(func(key, value interface{}) bool {
		orders, ok := value.([]common.Order)
		if !ok {
			err = fmt.Errorf("cannot convert value (%v) to Order", value)
			return false
		}
		result.Data = append(result.Data, orders...)
		return true
	})
	return result, err
}

func (self *Huobi) FetchEBalanceData(timepoint uint64) (common.EBalanceEntry, error) {
	result := common.EBalanceEntry{}
	result.Timestamp = common.Timestamp(fmt.Sprintf("%d", timepoint))
	result.Valid = true
	result.Error = ""
	resp_data, err := self.interf.GetInfo()
	result.ReturnTime = common.GetTimestamp()
	if err != nil {
		result.Valid = false
		result.Error = err.Error()
		result.Status = false
	} else {
		result.AvailableBalance = map[string]float64{}
		result.LockedBalance = map[string]float64{}
		result.DepositBalance = map[string]float64{}
		result.Status = true
		if resp_data.Status != "ok" {
			result.Valid = false
			result.Error = fmt.Sprintf("Cannot fetch ebalance")
			result.Status = false
		} else {
			balances := resp_data.Data.List
			for _, b := range balances {
				tokenID := strings.ToUpper(b.Currency)
				_, err := self.setting.GetInternalTokenByID(tokenID)
				if err == nil {
					balance, _ := strconv.ParseFloat(b.Balance, 64)
					if b.Type == "trade" {
						result.AvailableBalance[tokenID] = balance
					} else {
						result.LockedBalance[tokenID] = balance
					}
					result.DepositBalance[tokenID] = 0
				}
			}
		}
	}
	return result, nil
}

func (self *Huobi) FetchOnePairTradeHistory(
	wait *sync.WaitGroup,
	data *sync.Map,
	pair common.TokenPair) {

	defer wait.Done()
	result := []common.TradeHistory{}
	resp, err := self.interf.GetAccountTradeHistory(pair.Base, pair.Quote)
	if err != nil {
		log.Printf("Cannot fetch data for pair %s%s: %s", pair.Base.ID, pair.Quote.ID, err.Error())
	}
	pairString := pair.PairID()
	for _, trade := range resp.Data {
		price, _ := strconv.ParseFloat(trade.Price, 64)
		quantity, _ := strconv.ParseFloat(trade.Amount, 64)
		historyType := "sell"
		if trade.Type == "buy-limit" {
			historyType = "buy"
		}
		tradeHistory := common.NewTradeHistory(
			strconv.FormatUint(trade.ID, 10),
			price,
			quantity,
			historyType,
			trade.Timestamp,
		)
		result = append(result, tradeHistory)
	}
	data.Store(pairString, result)
}

func (self *Huobi) FetchTradeHistory() {
	t := time.NewTicker(10 * time.Minute)
	go func() {
		for {
			result := map[common.TokenPairID][]common.TradeHistory{}
			data := sync.Map{}
			pairs, err := self.TokenPairs()
			if err != nil {
				log.Printf("Huobi fetch trade history failed (%s). This might due to pairs setting hasn't been init yet", err.Error())
				continue
			}
			wait := sync.WaitGroup{}
			for _, pair := range pairs {
				wait.Add(1)
				go self.FetchOnePairTradeHistory(&wait, &data, pair)
			}
			wait.Wait()
			var integrity bool = true
			data.Range(func(key, value interface{}) bool {
				tokenPairID, ok := key.(common.TokenPairID)
				//if there is conversion error, continue to next key,val
				if !ok {
					log.Printf("Key (%v) cannot be asserted to TokenPairID", key)
					integrity = false
					return false
				}
				tradeHistories, ok := value.([]common.TradeHistory)
				if !ok {
					log.Printf("Value (%v) cannot be asserted to []TradeHistory", value)
					integrity = false
					return false
				}
				result[tokenPairID] = tradeHistories
				return true
			})
			if !integrity {
				log.Print("Huobi fetch trade history returns corrupted. Try again in 10 mins")
				continue
			}
			if err := self.storage.StoreTradeHistory(result); err != nil {
				log.Printf("Store trade history error: %s", err.Error())
			}
			<-t.C
		}
	}()
}

func (self *Huobi) GetTradeHistory(fromTime, toTime uint64) (common.ExchangeTradeHistory, error) {
	return self.storage.GetTradeHistory(fromTime, toTime)
}

func (self *Huobi) Send2ndTransaction(amount float64, token common.Token, exchangeAddress ethereum.Address) (*types.Transaction, error) {
	IAmount := common.FloatToBigInt(amount, token.Decimal)
	// Check balance, removed from huobi's blockchain object.
	// currBalance := self.blockchain.CheckBalance(token)
	// log.Printf("current balance of token %s is %d", token.ID, currBalance)
	// //self.blockchain.
	// if currBalance.Cmp(IAmount) < 0 {
	// 	log.Printf("balance is not enough, wait till next check")
	// 	return nil, errors.New("balance is not enough")
	// }
	var tx *types.Transaction
	var err error
	if token.ID == "ETH" {
		tx, err = self.blockchain.SendETHFromAccountToExchange(IAmount, exchangeAddress)
	} else {
		tx, err = self.blockchain.SendTokenFromAccountToExchange(IAmount, exchangeAddress, ethereum.HexToAddress(token.Address))
	}
	if err != nil {
		log.Printf("ERROR: Can not send transaction to exchange: %v", err)
		return nil, err
	}
	log.Printf("Transaction submitted. Tx is: %v", tx)
	return tx, nil

}

func (self *Huobi) PendingIntermediateTxs() (map[common.ActivityID]common.TXEntry, error) {
	result, err := self.storage.GetPendingIntermediateTXs()
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (self *Huobi) FindTx2InPending(id common.ActivityID) (common.TXEntry, bool) {
	pendings, err := self.storage.GetPendingIntermediateTXs()
	if err != nil {
		log.Printf("can't get pendings tx2 records: %v", err)
		return common.TXEntry{}, false
	}
	for actID, txentry := range pendings {
		if actID == id {
			return txentry, true
		}
	}
	return common.TXEntry{}, false
}

//FindTx2 : find Tx2 Record associates with activity ID, return
func (self *Huobi) FindTx2(id common.ActivityID) (Tx2 common.TXEntry, found bool) {
	found = true
	//first look it up in permanent bucket
	Tx2, err := self.storage.GetIntermedatorTx(id)
	if err != nil {
		//couldn't look for it in permanent bucket, look for it in pending bucket
		Tx2, found = self.FindTx2InPending(id)
	}
	return Tx2, found
}

func (self *Huobi) DepositStatus(id common.ActivityID, tx1Hash, currency string, sentAmount float64, timepoint uint64) (string, error) {

	var data common.TXEntry
	tx2Entry, found := self.FindTx2(id)
	if !found {
		//if not found, meaning there is no tx2 yet, check the 1st Tx to process.

		status, blockno, err := self.blockchain.TxStatus(ethereum.HexToHash(tx1Hash))
		if err != nil {
			log.Printf("Huobi Can not get TX status (%s)", err.Error())
			return "", nil
		}
		log.Printf("Huobi Status for Tx1 was %s at block %d ", status, blockno)
		if status == "mined" {
			//if it is mined, send 2nd tx.
			log.Printf("Found a new deposit status, which deposit %f %s. Procceed to send it to Huobi", sentAmount, currency)
			//check if the token is supported, the token can be active or inactivee
			token, err := self.setting.GetTokenByID(currency)
			if err != nil {
				return "", err
			}
			exchangeAddress, err := self.RealDepositAddress(currency)
			if err != nil {
				return "", err
			}
			tx2, err := self.Send2ndTransaction(sentAmount, token, exchangeAddress)
			if err != nil {
				log.Printf("Huobi Trying to send 2nd tx failed, error: %s. Will retry next time", err.Error())
				return "", nil
			}
			//store tx2 to pendingIntermediateTx
			data = common.NewTXEntry(
				tx2.Hash().Hex(),
				self.Name(),
				currency,
				"submitted",
				"",
				sentAmount,
				common.GetTimestamp(),
			)
			if err = self.storage.StorePendingIntermediateTx(id, data); err != nil {
				log.Printf("Trying to store 2nd tx to pending tx storage failed, error: %s. It will be ignored and can make us to send to huobi again and the deposit will be marked as failed because the fund is not efficient", err.Error())
			}
			return "", nil
		}
		//No need to handle other blockchain status of TX1 here, since Fetcher will handle it from blockchain Status.
		return "", nil
	}
	// if there is tx2Entry, check it blockchain status first:
	status, _, err := self.blockchain.TxStatus(ethereum.HexToHash(tx2Entry.Hash))
	if err != nil {
		return "", err
	}
	if status == "mined" {
		log.Println("Huobi 2nd Transaction is mined. Processed to store it and check the Huobi Deposit history")
		data = common.NewTXEntry(
			tx2Entry.Hash,
			self.Name(),
			currency,
			"mined",
			"",
			sentAmount,
			common.GetTimestamp(),
		)
		if err = self.storage.StorePendingIntermediateTx(id, data); err != nil {
			log.Printf("Huobi Trying to store intermediate tx to huobi storage, error: %s. Ignore it and try later", err.Error())
			return "", nil
		}
		tokens, uErr := self.setting.GetAllTokens()
		if uErr != nil {
			log.Printf("Huobi ERROR: Can not get list of tokens from setting (%s)", uErr)
			return "", uErr
		}
		var deposits HuobiDeposits
		deposits, err = self.interf.DepositHistory(tokens)
		if err != nil || deposits.Status != "ok" {
			log.Printf("Huobi Getting deposit history from huobi failed, error: %v, status: %s", err, deposits.Status)
			return "", nil
		}
		//check tx2 deposit status from Huobi
		for _, deposit := range deposits.Data {
			// log.Printf("deposit tx is %s, with token %s", deposit.TxHash, deposit.Currency)
			if deposit.TxHash == tx2Entry.Hash {
				if deposit.State == "safe" || deposit.State == "confirmed" {
					data = common.NewTXEntry(
						tx2Entry.Hash,
						self.Name(),
						currency,
						"mined",
						"done",
						sentAmount,
						common.GetTimestamp(),
					)
					if err = self.storage.StoreIntermediateTx(id, data); err != nil {
						log.Printf("Huobi Trying to store intermediate tx to huobi storage, error: %s. Ignore it and try later", err.Error())
						return "", nil
					}
					return "done", nil
				}
				//TODO : handle other states following https://github.com/huobiapi/API_Docs_en/wiki/REST_Reference#deposit-states
				log.Printf("Huobi Tx %s is found but the status was not safe but %s", deposit.TxHash, deposit.State)
				return "", nil
			}
		}
		log.Printf("Huobi Deposit doesn't exist. Huobi hasn't recognized the deposit yet or in theory, you have more than %d deposits at the same time.", len(tokens)*2)
		return "", nil
	} else if status == "failed" {
		data = common.NewTXEntry(
			tx2Entry.Hash,
			self.Name(),
			currency,
			"failed",
			"failed",
			sentAmount,
			common.GetTimestamp(),
		)

		return "failed", nil
	} else if status == "lost" {
		elapsed := common.GetTimepoint() - tx2Entry.Timestamp.ToUint64()
		if elapsed > uint64(15*time.Minute/time.Millisecond) {
			data = common.NewTXEntry(
				tx2Entry.Hash,
				self.Name(),
				currency,
				"lost",
				"lost",
				sentAmount,
				common.GetTimestamp(),
			)
			if err = self.storage.StoreIntermediateTx(id, data); err != nil {
				log.Printf("Huobi Trying to store intermediate tx failed, error: %s. Ignore it and treat it like it is still pending", err.Error())
				return "", nil
			}
			log.Printf("The tx is not found for over 15mins, it is considered as lost and the deposit failed")
			return "failed", nil
		}
		return "", nil
	}
	return "", nil
}

func (self *Huobi) WithdrawStatus(
	id, currency string, amount float64, timepoint uint64) (string, string, error) {
	withdrawID, _ := strconv.ParseUint(id, 10, 64)
	tokens, err := self.setting.GetAllTokens()
	if err != nil {
		return "", "", fmt.Errorf("Huobi Can't get list of token from setting (%s)", err)
	}
	withdraws, err := self.interf.WithdrawHistory(tokens)
	if err != nil {
		return "", "", nil
	}
	log.Printf("Huobi Withdrawal id: %d", withdrawID)
	for _, withdraw := range withdraws.Data {
		if withdraw.TxID == withdrawID {
			if withdraw.State == "confirmed" {
				return "done", withdraw.TxHash, nil
			}
			return "", withdraw.TxHash, nil
		}
	}
	return "", "", errors.New("Huobi Withdrawal doesn't exist. This shouldn't happen unless tx returned from withdrawal from huobi and activity ID are not consistently designed")
}

func (self *Huobi) OrderStatus(id string, base, quote string) (string, error) {
	orderID, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		return "", err
	}
	symbol := base + quote
	order, err := self.interf.OrderStatus(symbol, orderID)
	if err != nil {
		return "", err
	}
	if order.Data.State == "pre-submitted" || order.Data.State == "submitting" || order.Data.State == "submitted" || order.Data.State == "partial-filled" || order.Data.State == "partial-canceled" {
		return "", nil
	}
	return "done", nil
}

//NewHuobi creates new Huobi exchange instance
func NewHuobi(
	interf HuobiInterface,
	blockchain *blockchain.BaseBlockchain,
	signer blockchain.Signer,
	nonce blockchain.NonceCorpus,
	storage HuobiStorage,
	setting Setting) (*Huobi, error) {

	bc, err := huobiblockchain.NewBlockchain(blockchain, signer, nonce)
	if err != nil {
		return nil, err
	}

	huobiObj := Huobi{
		interf,
		bc,
		storage,
		setting,
	}
	huobiObj.FetchTradeHistory()
	huobiServer := huobihttp.NewHuobiHTTPServer(&huobiObj)
	go huobiServer.Run()
	return &huobiObj, nil
}
