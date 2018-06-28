package storage

import (
	"github.com/KyberNetwork/reserve-data/common"
	"github.com/boltdb/bolt"
)

const (
	TOKEN_BUCKET_BY_ID          string = "token_by_id"
	TOKEN_BUCKET_BY_ADDRESS     string = "token_by_addr"
	ADDRESS_SETTING_BUCKET      string = "address_setting"
	ADDRESS_SET_SETTING_BUCKET  string = "address_set_setting"
	EXCHANGE_FEE_BUCKET         string = "exchange_fee"
	EXCHANGE_MIN_DEPOSIT_BUCKET string = "exchange_min_deposit"
	EXCHANGE_DEPOSIT_ADDRESS    string = "exchange_deposit_address"
	EXCHANGE_TOKEN_PAIRS        string = "exchange_token_pairs"
	EXCHANGE_INFO               string = "exchange_info"
	EXCHANGE_STATUS             string = "exchange_status"
	EXCHANGE_NOTIFICATIONS      string = "exchange_notifications"
	PENDING_TOKEN_REQUEST       string = "pending_token_request"
)

type FilterFunction func(common.Token) bool

func isActive(t common.Token) bool {
	return t.Active
}

func isToken(_ common.Token) bool {
	return true
}

func isInternal(t common.Token) bool {
	return t.Active && t.Internal
}

func isExternal(t common.Token) bool {
	return t.Active && !t.Internal
}

type BoltSettingStorage struct {
	db *bolt.DB
}

func NewBoltSettingStorage(dbPath string) (*BoltSettingStorage, error) {
	var err error
	var db *bolt.DB
	db, err = bolt.Open(dbPath, 0600, nil)
	if err != nil {
		return nil, err
	}
	err = db.Update(func(tx *bolt.Tx) error {
		if _, uErr := tx.CreateBucketIfNotExists([]byte(TOKEN_BUCKET_BY_ID)); uErr != nil {
			return uErr
		}
		if _, uErr := tx.CreateBucketIfNotExists([]byte(TOKEN_BUCKET_BY_ADDRESS)); uErr != nil {
			return uErr
		}
		if _, uErr := tx.CreateBucketIfNotExists([]byte(ADDRESS_SETTING_BUCKET)); uErr != nil {
			return uErr
		}
		if _, uErr := tx.CreateBucketIfNotExists([]byte(ADDRESS_SET_SETTING_BUCKET)); uErr != nil {
			return uErr
		}
		if _, uErr := tx.CreateBucketIfNotExists([]byte(EXCHANGE_FEE_BUCKET)); uErr != nil {
			return uErr
		}
		if _, uErr := tx.CreateBucketIfNotExists([]byte(EXCHANGE_MIN_DEPOSIT_BUCKET)); uErr != nil {
			return uErr
		}
		if _, uErr := tx.CreateBucketIfNotExists([]byte(EXCHANGE_DEPOSIT_ADDRESS)); uErr != nil {
			return uErr
		}
		if _, uErr := tx.CreateBucketIfNotExists([]byte(EXCHANGE_TOKEN_PAIRS)); uErr != nil {
			return uErr
		}
		if _, uErr := tx.CreateBucketIfNotExists([]byte(EXCHANGE_INFO)); uErr != nil {
			return uErr
		}
		if _, uErr := tx.CreateBucketIfNotExists([]byte(EXCHANGE_STATUS)); uErr != nil {
			return uErr
		}
		if _, uErr := tx.CreateBucketIfNotExists([]byte(EXCHANGE_NOTIFICATIONS)); uErr != nil {
			return uErr
		}
		if _, uErr := tx.CreateBucketIfNotExists([]byte(PENDING_TOKEN_REQUEST)); uErr != nil {
			return uErr
		}
		return nil
	})
	if err != nil {
		panic(err)
	}
	storage := BoltSettingStorage{db}
	return &storage, nil
}
