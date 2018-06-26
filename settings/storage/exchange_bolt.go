package storage

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/KyberNetwork/reserve-data/boltutil"
	"github.com/KyberNetwork/reserve-data/common"
	"github.com/KyberNetwork/reserve-data/settings"
	"github.com/boltdb/bolt"
)

// GetFee returns a map[tokenID]exchangeFees and error if occur
func (boltSettingStorage *BoltSettingStorage) GetFee(ex settings.ExchangeName) (common.ExchangeFees, error) {
	var result common.ExchangeFees
	err := boltSettingStorage.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(EXCHANGE_FEE_BUCKET))
		if b == nil {
			return fmt.Errorf("bucket %s hasn't existed yet", EXCHANGE_FEE_BUCKET)
		}
		data := b.Get(boltutil.Uint64ToBytes(uint64(ex)))
		if data == nil {
			return fmt.Errorf("key %s hasn't existed yet", ex.String())
		}
		uErr := json.Unmarshal(data, &result)
		if uErr != nil {
			return uErr
		}
		return nil
	})
	return result, err
}

// StoreFee stores the fee with exchangeName as key into database and return error if occur
func (boltSettingStorage *BoltSettingStorage) StoreFee(ex settings.ExchangeName, data common.ExchangeFees) error {
	err := boltSettingStorage.db.Update(func(tx *bolt.Tx) error {
		return putFee(tx, ex, data)
	})
	return err
}

func putFee(tx *bolt.Tx, ex settings.ExchangeName, fee common.ExchangeFees) error {
	b, uErr := tx.CreateBucketIfNotExists([]byte(EXCHANGE_FEE_BUCKET))
	if uErr != nil {
		return uErr
	}
	dataJSON, uErr := json.Marshal(fee)
	if uErr != nil {
		return uErr
	}
	return b.Put(boltutil.Uint64ToBytes(uint64(ex)), dataJSON)
}

// GetMinDeposit returns a map[tokenID]MinDeposit and error if occur
func (boltSettingStorage *BoltSettingStorage) GetMinDeposit(ex settings.ExchangeName) (common.ExchangesMinDeposit, error) {
	result := make(common.ExchangesMinDeposit)
	err := boltSettingStorage.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(EXCHANGE_MIN_DEPOSIT_BUCKET))
		if b == nil {
			return fmt.Errorf("bucket %s hasn't existed yet", EXCHANGE_MIN_DEPOSIT_BUCKET)
		}
		data := b.Get(boltutil.Uint64ToBytes(uint64(ex)))
		if data == nil {
			return fmt.Errorf("key %s hasn't existed yet", ex.String())
		}
		uErr := json.Unmarshal(data, &result)
		if uErr != nil {
			return uErr
		}
		return nil
	})
	return result, err
}

// StoreMinDeposit stores the minDeposit with exchangeName as key into database and return error if occur
func (boltSettingStorage *BoltSettingStorage) StoreMinDeposit(ex settings.ExchangeName, data common.ExchangesMinDeposit) error {
	err := boltSettingStorage.db.Update(func(tx *bolt.Tx) error {
		return putMinDeposit(tx, ex, data)
	})
	return err
}

func putMinDeposit(tx *bolt.Tx, ex settings.ExchangeName, minDeposit common.ExchangesMinDeposit) error {
	b, uErr := tx.CreateBucketIfNotExists([]byte(EXCHANGE_MIN_DEPOSIT_BUCKET))
	if uErr != nil {
		return uErr
	}
	dataJSON, uErr := json.Marshal(minDeposit)
	if uErr != nil {
		return uErr
	}
	return b.Put(boltutil.Uint64ToBytes(uint64(ex)), dataJSON)
}

// GetDepositAddresses returns a map[tokenID]DepositAddress and error if occur
func (boltSettingStorage *BoltSettingStorage) GetDepositAddresses(ex settings.ExchangeName) (common.ExchangeAddresses, error) {
	result := make(common.ExchangeAddresses)
	err := boltSettingStorage.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(EXCHANGE_DEPOSIT_ADDRESS))
		if b == nil {
			return fmt.Errorf("bucket %s hasn't existed yet", EXCHANGE_DEPOSIT_ADDRESS)
		}
		data := b.Get(boltutil.Uint64ToBytes(uint64(ex)))
		if data == nil {
			return fmt.Errorf("key %s hasn't existed yet", ex.String())
		}
		uErr := json.Unmarshal(data, &result)
		if uErr != nil {
			return uErr
		}
		return nil
	})
	return result, err

}

func putDepositAddress(tx *bolt.Tx, ex settings.ExchangeName, addrs common.ExchangeAddresses) error {
	b, uErr := tx.CreateBucketIfNotExists([]byte(EXCHANGE_DEPOSIT_ADDRESS))
	if uErr != nil {
		return uErr
	}
	dataJSON, uErr := json.Marshal(addrs)
	if uErr != nil {
		return uErr
	}
	return b.Put(boltutil.Uint64ToBytes(uint64(ex)), dataJSON)
}

// StoreDepositAddress stores the depositAddress with exchangeName as key into database and
// return error if occur
func (boltSettingStorage *BoltSettingStorage) StoreDepositAddress(ex settings.ExchangeName, addrs common.ExchangeAddresses) error {
	err := boltSettingStorage.db.Update(func(tx *bolt.Tx) error {
		return putDepositAddress(tx, ex, addrs)
	})
	return err
}

// GetTokenPairs returns a list of TokenPairs available at current exchange
// return error if occur
func (boltSettingStorage *BoltSettingStorage) GetTokenPairs(ex settings.ExchangeName) ([]common.TokenPair, error) {
	var result []common.TokenPair
	err := boltSettingStorage.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(EXCHANGE_TOKEN_PAIRS))
		if b == nil {
			return fmt.Errorf("bucket %s hasn't existed yet", EXCHANGE_TOKEN_PAIRS)
		}
		data := b.Get(boltutil.Uint64ToBytes(uint64(ex)))
		if data == nil {
			return fmt.Errorf("key %s hasn't existed yet", ex.String())
		}
		if uErr := json.Unmarshal(data, &result); uErr != nil {
			return uErr
		}
		return nil
	})
	return result, err
}

// StoreTokenPairs store the list of TokenPairs with exchangeName as key into database and
// return error if occur
func (boltSettingStorage *BoltSettingStorage) StoreTokenPairs(ex settings.ExchangeName, data []common.TokenPair) error {
	err := boltSettingStorage.db.Update(func(tx *bolt.Tx) error {
		b, uErr := tx.CreateBucketIfNotExists([]byte(EXCHANGE_TOKEN_PAIRS))
		if uErr != nil {
			return uErr
		}
		dataJSON, uErr := json.Marshal(data)
		if uErr != nil {
			return uErr
		}
		return b.Put(boltutil.Uint64ToBytes(uint64(ex)), dataJSON)
	})
	return err
}

func (boltSettingStorage *BoltSettingStorage) GetExchangeInfo(ex settings.ExchangeName) (common.ExchangeInfo, error) {
	var result common.ExchangeInfo
	err := boltSettingStorage.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(EXCHANGE_INFO))
		if b == nil {
			return fmt.Errorf("bucket %s hasn't existed yet", EXCHANGE_INFO)
		}
		data := b.Get(boltutil.Uint64ToBytes(uint64(ex)))
		if data == nil {
			return fmt.Errorf("key %s hasn't existed yet", ex.String())
		}
		return json.Unmarshal(data, &result)
	})
	return result, err
}

func putExchangeInfo(tx *bolt.Tx, ex settings.ExchangeName, exInfo common.ExchangeInfo) error {
	b, uErr := tx.CreateBucketIfNotExists([]byte(EXCHANGE_INFO))
	if uErr != nil {
		return uErr
	}
	dataJSON, uErr := json.Marshal(exInfo)
	if uErr != nil {
		return uErr
	}
	return b.Put(boltutil.Uint64ToBytes(uint64(ex)), dataJSON)
}

func (boltSettingStorage *BoltSettingStorage) StoreExchangeInfo(ex settings.ExchangeName, exInfo common.ExchangeInfo) error {
	err := boltSettingStorage.db.Update(func(tx *bolt.Tx) error {
		return putExchangeInfo(tx, ex, exInfo)
	})
	return err
}

// GetExchangeStatus get exchange status to dashboard and analytics
func (boltSettingStorage *BoltSettingStorage) GetExchangeStatus() (common.ExchangesStatus, error) {
	result := make(common.ExchangesStatus)
	var err error
	err = boltSettingStorage.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(EXCHANGE_STATUS))
		if b == nil {
			return fmt.Errorf("Bucket %s hasn't existed yet", EXCHANGE_STATUS)
		}
		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var exstat common.ExStatus
			if _, vErr := common.GetExchange(string(k)); vErr != nil {
				continue
			}
			if vErr := json.Unmarshal(v, &exstat); vErr != nil {
				return vErr
			}
			result[string(k)] = exstat
		}
		return nil
	})
	return result, err
}

func (boltSettingStorage *BoltSettingStorage) StoreExchangeStatus(data common.ExchangesStatus) error {
	var err error
	err = boltSettingStorage.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(EXCHANGE_STATUS))
		if b == nil {
			return fmt.Errorf("Bucket %s hasn't existed yet", EXCHANGE_STATUS)
		}
		for k, v := range data {
			dataJSON, uErr := json.Marshal(v)
			if uErr != nil {
				return uErr
			}
			if uErr := b.Put([]byte(k), dataJSON); uErr != nil {
				return uErr
			}
		}
		return nil
	})
	return err
}

func (boltSettingStorage *BoltSettingStorage) StoreExchangeNotification(
	exchange, action, token string, fromTime, toTime uint64, isWarning bool, msg string) error {
	var err error
	err = boltSettingStorage.db.Update(func(tx *bolt.Tx) error {
		exchangeBk := tx.Bucket([]byte(EXCHANGE_NOTIFICATIONS))
		b, uErr := exchangeBk.CreateBucketIfNotExists([]byte(exchange))
		if uErr != nil {
			return uErr
		}
		key := fmt.Sprintf("%s_%s", action, token)
		noti := common.ExchangeNotiContent{
			FromTime:  fromTime,
			ToTime:    toTime,
			IsWarning: isWarning,
			Message:   msg,
		}

		// update new value
		dataJSON, uErr := json.Marshal(noti)
		if uErr != nil {
			return uErr
		}
		return b.Put([]byte(key), dataJSON)
	})
	return err
}

func (boltSettingStorage *BoltSettingStorage) GetExchangeNotifications() (common.ExchangeNotifications, error) {
	result := common.ExchangeNotifications{}
	var err error
	err = boltSettingStorage.db.View(func(tx *bolt.Tx) error {
		exchangeBks := tx.Bucket([]byte(EXCHANGE_NOTIFICATIONS))
		c := exchangeBks.Cursor()
		for name, bucket := c.First(); name != nil; name, bucket = c.Next() {
			// if bucket == nil, then name is a child bucket name (according to bolt docs)
			if bucket == nil {
				b := exchangeBks.Bucket(name)
				c := b.Cursor()
				actionContent := common.ExchangeActionNoti{}
				for k, v := c.First(); k != nil; k, v = c.Next() {
					actionToken := strings.Split(string(k), "_")
					action := actionToken[0]
					token := actionToken[1]
					notiContent := common.ExchangeNotiContent{}
					if uErr := json.Unmarshal(v, &notiContent); uErr != nil {
						return uErr
					}
					tokenContent, exist := actionContent[action]
					if !exist {
						tokenContent = common.ExchangeTokenNoti{}
					}
					tokenContent[token] = notiContent
					actionContent[action] = tokenContent
				}
				result[string(name)] = actionContent
			}
		}
		return nil
	})
	return result, err
}
