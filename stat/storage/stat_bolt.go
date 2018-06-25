package storage

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/KyberNetwork/reserve-data/boltutil"
	"github.com/KyberNetwork/reserve-data/common"
	"github.com/boltdb/bolt"
	ethereum "github.com/ethereum/go-ethereum/common"
)

const (
	MAX_NUMBER_VERSION int = 1000

	TRADELOG_PROCESSOR_STATE string = "tradelog_processor_state"

	MINUTE_BUCKET               string = "minute"
	HOUR_BUCKET                 string = "hour"
	DAY_BUCKET                  string = "day"
	TIMEZONE_BUCKET_PREFIX      string = "utc"
	START_TIMEZONE              int64  = -11
	END_TIMEZONE                int64  = 14
	ADDRESS_BUCKET_PREFIX       string = "address"
	DAILY_ADDRESS_BUCKET_PREFIX string = "daily_address"
	DAILY_USER_BUCKET_PREFIX    string = "daily_user"
	WALLET_ADDRESS_BUCKET       string = "wallet_address"
	RESERVE_RATES               string = "reserve_rates"
	EXPIRED                     uint64 = uint64(7 * time.Hour * 24)
	COUNTRY_BUCKET              string = "country_stat_bucket"
	USER_FIRST_TRADE_EVER       string = "user_first_trade_ever"
	USER_STAT_BUCKET            string = "user_stat_bucket"
	VOLUME_STAT_BUCKET          string = "volume_stat_bucket"
	USER_LIST_BUCKET            string = "user_list"

	TRADE_SUMMARY_AGGREGATION string = "trade_summary_aggregation"
	WALLET_AGGREGATION        string = "wallet_aggregation"
	COUNTRY_AGGREGATION       string = "country_aggregation"
	USER_AGGREGATION          string = "user_aggregation"
	VOLUME_STAT_AGGREGATION   string = "volume_stat_aggregation"
	BURNFEE_AGGREGATION       string = "burn_fee_aggregation"
	USER_INFO_AGGREGATION     string = "user_info_aggregation"
)

//BoltStatStorage object
type BoltStatStorage struct {
	db *bolt.DB
}

//NewBoltStatStorage return new storage instance
func NewBoltStatStorage(path string) (*BoltStatStorage, error) {
	// init instance
	var err error
	var db *bolt.DB
	db, err = bolt.Open(path, 0600, nil)
	if err != nil {
		return nil, err
	}
	// init buckets
	err = db.Update(func(tx *bolt.Tx) error {
		_, err = tx.CreateBucketIfNotExists([]byte(TRADELOG_PROCESSOR_STATE))
		if err != nil {
			return err
		}
		_, err = tx.CreateBucketIfNotExists([]byte(WALLET_ADDRESS_BUCKET))
		if err != nil {
			return err
		}
		_, err = tx.CreateBucketIfNotExists([]byte(COUNTRY_BUCKET))
		if err != nil {
			return err
		}

		return err
	})
	if err != nil {
		return nil, err
	}
	storage := &BoltStatStorage{db}
	return storage, nil
}

func (self *BoltStatStorage) SetLastProcessedTradeLogTimepoint(statType string, timepoint uint64) error {
	var err error
	err = self.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(TRADELOG_PROCESSOR_STATE))
		err = b.Put([]byte(statType), boltutil.Uint64ToBytes(timepoint))
		return err
	})
	return err
}

func (self *BoltStatStorage) GetLastProcessedTradeLogTimepoint(statType string) (uint64, error) {
	var result uint64
	var err error
	err = self.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(TRADELOG_PROCESSOR_STATE))
		if b == nil {
			return (errors.New("Can not find last processed bucket"))
		}
		result = boltutil.BytesToUint64(b.Get([]byte(statType)))
		return nil
	})
	return result, err
}

// GetNumberOfVersion return number of version storing in a bucket
func (self *BoltStatStorage) GetNumberOfVersion(tx *bolt.Tx, bucket string) int {
	result := 0
	b := tx.Bucket([]byte(bucket))
	c := b.Cursor()
	for k, _ := c.First(); k != nil; k, _ = c.Next() {
		result++
	}
	return result
}

// PruneOutdatedData Remove first version out of database
func (self *BoltStatStorage) PruneOutdatedData(tx *bolt.Tx, bucket string) error {
	var err error
	b := tx.Bucket([]byte(bucket))
	c := b.Cursor()
	for self.GetNumberOfVersion(tx, bucket) >= MAX_NUMBER_VERSION {
		k, _ := c.First()
		if k == nil {
			err = fmt.Errorf("There no version in %s", bucket)
			return err
		}
		err = b.Delete([]byte(k))
		if err != nil {
			panic(err)
		}
	}
	return err
}

func getBucketNameByFreq(freq string) (bucketName string, err error) {
	switch freq {
	case "m", "M":
		bucketName = MINUTE_BUCKET
	case "h", "H":
		bucketName = HOUR_BUCKET
	case "d", "D":
		bucketName = DAY_BUCKET
	default:
		offset, ok := strconv.ParseInt(strings.TrimPrefix(freq, "utc"), 10, 64)
		if (offset < START_TIMEZONE) || (offset > END_TIMEZONE) {
			err = errors.New("Frequency is wrong, can not get bucket name")
		}
		if ok != nil {
			err = ok
		}
		bucketName = freq
	}
	return
}

func getTimestampByFreq(t uint64, freq string) (result []byte) {
	ui64Day := uint64(time.Hour * 24)
	switch freq {
	case "m", "M":
		result = boltutil.Uint64ToBytes(t / uint64(time.Minute) * uint64(time.Minute))
	case "h", "H":
		result = boltutil.Uint64ToBytes(t / uint64(time.Hour) * uint64(time.Hour))
	case "d", "D":
		result = boltutil.Uint64ToBytes(t / ui64Day * ui64Day)
	default:
		// utc timezone
		offset, _ := strconv.ParseInt(strings.TrimPrefix(freq, "utc"), 10, 64)
		ui64offset := uint64(int64(time.Hour) * offset)
		if offset > 0 {
			result = boltutil.Uint64ToBytes((t+ui64offset)/ui64Day*ui64Day + ui64offset)
		} else {
			offset = 0 - offset
			result = boltutil.Uint64ToBytes((t-ui64offset)/ui64Day*ui64Day - ui64offset)
		}
	}

	return
}

func (self *BoltStatStorage) SetWalletAddress(ethWalletAddr ethereum.Address) (err error) {
	walletAddr := common.AddrToString(ethWalletAddr)
	err = self.db.Update(func(tx *bolt.Tx) error {
		walletBucket := tx.Bucket([]byte(WALLET_ADDRESS_BUCKET))

		if err := walletBucket.Put([]byte(walletAddr), []byte("1")); err != nil {
			return err
		}

		return nil
	})
	return
}

func (self *BoltStatStorage) GetWalletAddress() ([]string, error) {
	var result []string
	var err error
	err = self.db.View(func(tx *bolt.Tx) error {
		walletBucket := tx.Bucket([]byte(WALLET_ADDRESS_BUCKET))
		c := walletBucket.Cursor()
		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			result = append(result, string(k[:]))
		}
		return nil
	})
	return result, err
}

func (self *BoltStatStorage) SetBurnFeeStat(burnFeeStats map[string]common.BurnFeeStatsTimeZone, lastProcessTimePoint uint64) error {
	err := self.db.Update(func(tx *bolt.Tx) error {
		for key, timezoneData := range burnFeeStats {
			key = strings.ToLower(key)
			burnFeeBk, _ := tx.CreateBucketIfNotExists([]byte(key))
			for _, freq := range []string{"M", "H", "D"} {
				stats := timezoneData[freq]
				freqBkName, _ := getBucketNameByFreq(freq)
				freqBk, _ := burnFeeBk.CreateBucketIfNotExists([]byte(freqBkName))
				for timepoint, stat := range stats {
					timestamp := boltutil.Uint64ToBytes(timepoint)
					currentData := common.BurnFeeStats{}
					v := freqBk.Get(timestamp)
					if v != nil {
						if err := json.Unmarshal(v, &currentData); err != nil {
							log.Printf("Unmarshal current data error: %s", err.Error())
							return err
						}
					}
					currentData.TotalBurnFee += stat.TotalBurnFee

					dataJSON, _ := json.Marshal(currentData)
					if err := freqBk.Put(timestamp, dataJSON); err != nil {
						return err
					}
				}
			}
		}
		lastProcessBk := tx.Bucket([]byte(TRADELOG_PROCESSOR_STATE))
		if lastProcessBk != nil {
			dataJSON := boltutil.Uint64ToBytes(lastProcessTimePoint)
			return lastProcessBk.Put([]byte(BURNFEE_AGGREGATION), dataJSON)
		}
		return nil
	})
	return err
}

func (self *BoltStatStorage) SetVolumeStat(volumeStats map[string]common.VolumeStatsTimeZone, lastProcessTimePoint uint64) error {
	err := self.db.Update(func(tx *bolt.Tx) error {
		for asset, freqData := range volumeStats {
			asset = strings.ToLower(asset)
			volumeBk, _ := tx.CreateBucketIfNotExists([]byte(asset))
			for _, freq := range []string{"M", "H", "D"} {
				stats := freqData[freq]
				freqBkName, _ := getBucketNameByFreq(freq)
				freqBk, _ := volumeBk.CreateBucketIfNotExists([]byte(freqBkName))
				for timepoint, stat := range stats {
					timestamp := boltutil.Uint64ToBytes(timepoint)
					currentData := common.VolumeStats{}
					v := freqBk.Get(timestamp)
					if v != nil {
						if err := json.Unmarshal(v, &currentData); err != nil {
							return err
						}
					}
					currentData.ETHVolume += stat.ETHVolume
					currentData.USDAmount += stat.USDAmount
					currentData.Volume += stat.Volume

					dataJSON, _ := json.Marshal(currentData)
					if err := freqBk.Put(timestamp, dataJSON); err != nil {
						return err
					}
				}
			}
		}
		lastProcessBk := tx.Bucket([]byte(TRADELOG_PROCESSOR_STATE))
		if lastProcessBk != nil {
			dataJSON := boltutil.Uint64ToBytes(lastProcessTimePoint)
			return lastProcessBk.Put([]byte(VOLUME_STAT_AGGREGATION), dataJSON)
		}
		return nil
	})
	return err
}

func (self *BoltStatStorage) SetWalletStat(stats map[string]common.MetricStatsTimeZone, lastProcessTimePoint uint64) error {
	err := self.db.Update(func(tx *bolt.Tx) error {
		for wallet, timeZoneStat := range stats {
			wallet = strings.ToLower(wallet)
			b, err := tx.CreateBucketIfNotExists([]byte(wallet))
			if err != nil {
				return err
			}
			for i := START_TIMEZONE; i <= END_TIMEZONE; i++ {
				stats := timeZoneStat[i]
				freq := fmt.Sprintf("%s%d", TIMEZONE_BUCKET_PREFIX, i)
				walletTzBucket, err := b.CreateBucketIfNotExists([]byte(freq))
				if err != nil {
					return err
				}
				for timepoint, stat := range stats {
					timestamp := boltutil.Uint64ToBytes(timepoint)
					currentData := common.MetricStats{}
					v := walletTzBucket.Get(timestamp)
					if v != nil {
						if err := json.Unmarshal(v, &currentData); err != nil {
							return err
						}
					}
					currentData.ETHVolume += stat.ETHVolume
					currentData.USDVolume += stat.USDVolume
					currentData.BurnFee += stat.BurnFee
					currentData.TradeCount += stat.TradeCount
					currentData.UniqueAddr += stat.UniqueAddr
					currentData.NewUniqueAddresses += stat.NewUniqueAddresses
					currentData.KYCEd += stat.KYCEd
					if currentData.TradeCount > 0 {
						currentData.ETHPerTrade = currentData.ETHVolume / float64(currentData.TradeCount)
						currentData.USDPerTrade = currentData.USDVolume / float64(currentData.TradeCount)
					}
					dataJSON, err := json.Marshal(currentData)
					if err != nil {
						return err
					}
					if err := walletTzBucket.Put(timestamp, dataJSON); err != nil {
						return err
					}
				}
			}
		}
		lastProcessBk := tx.Bucket([]byte(TRADELOG_PROCESSOR_STATE))
		if lastProcessBk != nil {
			dataJSON := boltutil.Uint64ToBytes(lastProcessTimePoint)
			return lastProcessBk.Put([]byte(WALLET_AGGREGATION), dataJSON)
		}
		return nil
	})
	return err
}

func (self *BoltStatStorage) GetWalletStats(fromTime uint64, toTime uint64, ethWalletAddr ethereum.Address, timezone int64) (common.StatTicks, error) {
	walletAddr := common.AddrToString(ethWalletAddr)
	result := common.StatTicks{}
	tzstring := fmt.Sprintf("%s%d", TIMEZONE_BUCKET_PREFIX, timezone)
	err := self.db.Update(func(tx *bolt.Tx) error {
		walletBk, _ := tx.CreateBucketIfNotExists([]byte(walletAddr))
		timezoneBk, _ := walletBk.CreateBucketIfNotExists([]byte(tzstring))
		min := boltutil.Uint64ToBytes(fromTime)
		max := boltutil.Uint64ToBytes(toTime)
		c := timezoneBk.Cursor()
		for k, v := c.Seek(min); k != nil && bytes.Compare(k, max) <= 0; k, v = c.Next() {
			walletStat := common.MetricStats{}
			if err := json.Unmarshal(v, &walletStat); err != nil {
				log.Printf("Unmarshal wallet stat error: %s", err.Error())
				return err
			}
			key := boltutil.BytesToUint64(k) / 1000000
			result[key] = walletStat
		}
		return nil
	})
	return result, err
}

func (self *BoltStatStorage) SetCountry(country string) error {
	var err error
	country = strings.ToUpper(country)
	err = self.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(COUNTRY_BUCKET))
		err = b.Put([]byte(country), []byte("1"))
		return err
	})
	return err
}

func (self *BoltStatStorage) GetCountries() ([]string, error) {
	countries := []string{}
	var err error
	err = self.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(COUNTRY_BUCKET))
		c := b.Cursor()
		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			countries = append(countries, string(k))
		}
		return err
	})
	return countries, err
}

func (self *BoltStatStorage) SetCountryStat(stats map[string]common.MetricStatsTimeZone, lastProcessTimePoint uint64) error {
	var err error
	err = self.db.Update(func(tx *bolt.Tx) error {
		for country, timeZoneStat := range stats {
			country = strings.ToUpper(country)
			b, uErr := tx.CreateBucketIfNotExists([]byte(country))
			if uErr != nil {
				return uErr
			}
			for i := START_TIMEZONE; i <= END_TIMEZONE; i++ {
				var (
					uErr     error
					dataJSON []byte
				)
				stats := timeZoneStat[i]
				freq := fmt.Sprintf("%s%d", TIMEZONE_BUCKET_PREFIX, i)
				countryTzBucket, uErr := b.CreateBucketIfNotExists([]byte(freq))
				if uErr != nil {
					return uErr
				}
				for timepoint, stat := range stats {
					timestamp := boltutil.Uint64ToBytes(timepoint)
					currentData := common.MetricStats{}
					v := countryTzBucket.Get(timestamp)
					if v != nil {
						if uErr = json.Unmarshal(v, &currentData); uErr != nil {
							return uErr
						}
					}
					currentData.ETHVolume += stat.ETHVolume
					currentData.USDVolume += stat.USDVolume
					currentData.BurnFee += stat.BurnFee
					currentData.TradeCount += stat.TradeCount
					currentData.UniqueAddr += stat.UniqueAddr
					currentData.NewUniqueAddresses += stat.NewUniqueAddresses
					currentData.KYCEd += stat.KYCEd
					if currentData.TradeCount > 0 {
						currentData.ETHPerTrade = currentData.ETHVolume / float64(currentData.TradeCount)
						currentData.USDPerTrade = currentData.USDVolume / float64(currentData.TradeCount)
					}

					if dataJSON, uErr = json.Marshal(currentData); uErr != nil {
						return uErr
					}
					if uErr = countryTzBucket.Put(timestamp, dataJSON); uErr != nil {
						return uErr
					}
				}
			}
		}
		lastProcessBk := tx.Bucket([]byte(TRADELOG_PROCESSOR_STATE))
		if lastProcessBk != nil {
			dataJSON := boltutil.Uint64ToBytes(lastProcessTimePoint)
			return lastProcessBk.Put([]byte(COUNTRY_AGGREGATION), dataJSON)
		}
		return nil
	})
	return err
}

func (self *BoltStatStorage) GetCountryStats(fromTime, toTime uint64, country string, timezone int64) (common.StatTicks, error) {
	result := common.StatTicks{}
	tzstring := fmt.Sprintf("%s%d", TIMEZONE_BUCKET_PREFIX, timezone)
	country = strings.ToUpper(country)
	err := self.db.Update(func(tx *bolt.Tx) error {
		countryBk, _ := tx.CreateBucketIfNotExists([]byte(country))
		timezoneBk, _ := countryBk.CreateBucketIfNotExists([]byte(tzstring))
		min := boltutil.Uint64ToBytes(fromTime)
		max := boltutil.Uint64ToBytes(toTime)

		c := timezoneBk.Cursor()
		for k, v := c.Seek(min); k != nil && bytes.Compare(k, max) <= 0; k, v = c.Next() {
			countryStat := common.MetricStats{}
			if err := json.Unmarshal(v, &countryStat); err != nil {
				return err
			}
			key := boltutil.BytesToUint64(k) / 1000000
			result[key] = countryStat
		}
		return nil
	})

	return result, err
}

func (self *BoltStatStorage) DidTrade(tx *bolt.Tx, userAddr string, timepoint uint64) bool {
	result := false
	b, _ := tx.CreateBucketIfNotExists([]byte(USER_FIRST_TRADE_EVER))
	v := b.Get([]byte(userAddr))
	if v != nil {
		savedTimepoint := boltutil.BytesToUint64(v)
		if savedTimepoint <= timepoint {
			result = true
		}
	}
	return result
}

func (self *BoltStatStorage) SetFirstTradeEver(userTradeLog *[]common.TradeLog) error {
	err := self.db.Update(func(tx *bolt.Tx) error {
		b, _ := tx.CreateBucketIfNotExists([]byte(USER_FIRST_TRADE_EVER))
		for _, trade := range *userTradeLog {
			userAddr := common.AddrToString(trade.UserAddress)
			timepoint := trade.Timestamp
			if !self.DidTrade(tx, userAddr, timepoint) {
				timestampByte := boltutil.Uint64ToBytes(timepoint)
				if pErr := b.Put([]byte(userAddr), timestampByte); pErr != nil {
					log.Printf("Cannot put data: %s", pErr.Error())
				}
			}
		}
		return nil
	})
	return err
}

func (self *BoltStatStorage) GetFirstTradeEver(ethUserAddr ethereum.Address) (uint64, error) {
	result := uint64(0)
	userAddr := common.AddrToString(ethUserAddr)
	err := self.db.Update(func(tx *bolt.Tx) error {
		b, _ := tx.CreateBucketIfNotExists([]byte(USER_FIRST_TRADE_EVER))
		v := b.Get([]byte(userAddr))
		result = boltutil.BytesToUint64(v)
		return nil
	})
	return result, err
}

func (self *BoltStatStorage) GetAllFirstTradeEver() (map[ethereum.Address]uint64, error) {
	result := map[ethereum.Address]uint64{}
	err := self.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(USER_FIRST_TRADE_EVER))
		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			value := boltutil.BytesToUint64(v)
			result[ethereum.HexToAddress(string(k))] = value
		}
		return nil
	})
	return result, err
}

func (self *BoltStatStorage) DidTradeInDay(tx *bolt.Tx, userAddr string, timepoint uint64, timezone int64) bool {
	result := false
	userStatBk, _ := tx.CreateBucketIfNotExists([]byte(USER_STAT_BUCKET))
	freq := fmt.Sprintf("%s%d", TIMEZONE_BUCKET_PREFIX, timezone)
	timestamp := getTimestampByFreq(timepoint, freq)

	timezoneBk, _ := userStatBk.CreateBucketIfNotExists(boltutil.Uint64ToBytes(uint64(timezone)))
	userDailyBucket, _ := timezoneBk.CreateBucketIfNotExists(timestamp)

	v := userDailyBucket.Get([]byte(userAddr))
	if v != nil {
		savedTimepoint := boltutil.BytesToUint64(v)
		if savedTimepoint <= timepoint {
			result = true
		}
	}
	return result
}

func (self *BoltStatStorage) GetFirstTradeInDay(ethUserAddr ethereum.Address, timepoint uint64, timezone int64) (uint64, error) {
	result := uint64(0)
	userAddr := common.AddrToString(ethUserAddr)
	err := self.db.Update(func(tx *bolt.Tx) error {
		userStatBk, _ := tx.CreateBucketIfNotExists([]byte(USER_STAT_BUCKET))
		freq := fmt.Sprintf("%s%d", TIMEZONE_BUCKET_PREFIX, timezone)
		timestamp := getTimestampByFreq(timepoint, freq)

		timezoneBk, _ := userStatBk.CreateBucketIfNotExists(boltutil.Uint64ToBytes(uint64(timezone)))
		userDailyBucket, _ := timezoneBk.CreateBucketIfNotExists(timestamp)

		v := userDailyBucket.Get([]byte(userAddr))
		if v != nil {
			result = boltutil.BytesToUint64(v)
		}
		return nil
	})
	return result, err
}

func (self *BoltStatStorage) SetFirstTradeInDay(tradeLogs *[]common.TradeLog) error {
	err := self.db.Update(func(tx *bolt.Tx) error {
		userStatBk, _ := tx.CreateBucketIfNotExists([]byte(USER_STAT_BUCKET))
		for _, trade := range *tradeLogs {
			userAddr := common.AddrToString(trade.UserAddress)
			timepoint := trade.Timestamp
			for timezone := START_TIMEZONE; timezone <= END_TIMEZONE; timezone++ {
				freq := fmt.Sprintf("%s%d", TIMEZONE_BUCKET_PREFIX, timezone)
				timestamp := getTimestampByFreq(timepoint, freq)
				timezoneBk, _ := userStatBk.CreateBucketIfNotExists(boltutil.Uint64ToBytes(uint64(timezone)))
				userDailyBucket, _ := timezoneBk.CreateBucketIfNotExists(timestamp)
				if !self.DidTradeInDay(tx, userAddr, timepoint, timezone) {
					timestampByte := boltutil.Uint64ToBytes(timepoint)
					if pErr := userDailyBucket.Put([]byte(userAddr), timestampByte); pErr != nil {
						log.Printf("Cannot put user daily first trade: %s", pErr.Error())
					}
				}
			}
		}
		return nil
	})
	return err
}

func (self *BoltStatStorage) SetUserList(userInfos map[string]common.UserInfoTimezone, lastProcessTimePoint uint64) error {
	err := self.db.Update(func(tx *bolt.Tx) error {
		b, uErr := tx.CreateBucketIfNotExists([]byte(USER_LIST_BUCKET))
		if uErr != nil {
			return uErr
		}
		for userAddr, userInfoData := range userInfos {
			var (
				timezoneBk *bolt.Bucket
				userAddr   = strings.ToLower(userAddr)
			)
			for timezone := START_TIMEZONE; timezone <= END_TIMEZONE; timezone++ {
				freq := fmt.Sprintf("%s%d", TIMEZONE_BUCKET_PREFIX, timezone)
				if timezoneBk, uErr = b.CreateBucketIfNotExists([]byte(freq)); uErr != nil {
					return uErr
				}
				timezoneData := userInfoData[timezone]
				for timepoint, userData := range timezoneData {
					var timestampBk *bolt.Bucket
					timestamp := getTimestampByFreq(timepoint, freq)
					if timestampBk, uErr = timezoneBk.CreateBucketIfNotExists(timestamp); uErr != nil {
						return uErr
					}
					currentUserData := common.UserInfo{}
					currentValue := timestampBk.Get([]byte(userAddr))
					if currentValue != nil {
						if uErr = json.Unmarshal(currentValue, &currentUserData); uErr != nil {
							return uErr
						}
					}
					currentUserData.USDVolume += userData.USDVolume
					currentUserData.ETHVolume += userData.ETHVolume
					currentUserData.Addr = userData.Addr
					currentUserData.Email = userData.Email
					dataJSON, _ := json.Marshal(currentUserData)
					if uErr = timestampBk.Put([]byte(userAddr), dataJSON); uErr != nil {
						return uErr
					}
				}
			}
		}
		lastProcessBk := tx.Bucket([]byte(TRADELOG_PROCESSOR_STATE))
		if lastProcessBk != nil {
			dataJSON := boltutil.Uint64ToBytes(lastProcessTimePoint)
			return lastProcessBk.Put([]byte(USER_INFO_AGGREGATION), dataJSON)
		}
		return nil
	})
	return err
}

func (self *BoltStatStorage) GetUserList(fromTime, toTime uint64, timezone int64) (map[string]common.UserInfo, error) {
	result := map[string]common.UserInfo{}
	err := self.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(USER_LIST_BUCKET))
		min := boltutil.Uint64ToBytes(fromTime)
		max := boltutil.Uint64ToBytes(toTime)

		timezoneBkName := fmt.Sprintf("%s%d", TIMEZONE_BUCKET_PREFIX, timezone)
		timezoneBk, err := b.CreateBucketIfNotExists([]byte(timezoneBkName))
		if err != nil {
			return err
		}
		c := timezoneBk.Cursor()
		for k, v := c.Seek(min); k != nil && bytes.Compare(k, max) <= 0; k, v = c.Next() {
			if v == nil {
				timestampBk, _ := timezoneBk.CreateBucketIfNotExists(k)
				cursor := timestampBk.Cursor()
				for kk, vv := cursor.First(); kk != nil; kk, vv = cursor.Next() {
					value := common.UserInfo{}
					uErr := json.Unmarshal(vv, &value)
					if uErr != nil {
						return uErr
					}
					currentData, exist := result[value.Addr]
					if !exist {
						currentData = common.UserInfo{
							Email: value.Email,
							Addr:  value.Addr,
						}
					}
					currentData.ETHVolume += value.ETHVolume
					currentData.USDVolume += value.USDVolume
					result[value.Addr] = currentData
				}
			}
		}
		return err
	})
	return result, err
}

func (self *BoltStatStorage) GetAssetVolume(fromTime uint64, toTime uint64, freq string, ethAssetAddr ethereum.Address) (common.StatTicks, error) {
	result := common.StatTicks{}
	assetAddr := common.AddrToString(ethAssetAddr)
	err := self.db.Update(func(tx *bolt.Tx) error {
		b, _ := tx.CreateBucketIfNotExists([]byte(assetAddr))

		freqBkName, _ := getBucketNameByFreq(freq)
		min := boltutil.Uint64ToBytes(fromTime)
		max := boltutil.Uint64ToBytes(toTime)

		freqBk, _ := b.CreateBucketIfNotExists([]byte(freqBkName))
		c := freqBk.Cursor()
		for k, v := c.Seek(min); k != nil && bytes.Compare(k, max) <= 0; k, v = c.Next() {
			value := common.VolumeStats{}
			if err := json.Unmarshal(v, &value); err != nil {
				return err
			}
			key := boltutil.BytesToUint64(k) / 1000000
			result[key] = value
		}

		return nil
	})
	return result, err
}

func (self *BoltStatStorage) GetBurnFee(fromTime uint64, toTime uint64, freq string, ethReserveAddr ethereum.Address) (common.StatTicks, error) {
	result := common.StatTicks{}
	reserveAddr := common.AddrToString(ethReserveAddr)
	err := self.db.Update(func(tx *bolt.Tx) error {
		b, _ := tx.CreateBucketIfNotExists([]byte(reserveAddr))
		freqBkName, _ := getBucketNameByFreq(freq)

		freqBk, _ := b.CreateBucketIfNotExists([]byte(freqBkName))

		min := boltutil.Uint64ToBytes(fromTime)
		max := boltutil.Uint64ToBytes(toTime)

		c := freqBk.Cursor()
		for k, v := c.Seek(min); k != nil && bytes.Compare(k, max) <= 0; k, v = c.Next() {
			value := common.BurnFeeStats{}
			if err := json.Unmarshal(v, &value); err != nil {
				return err
			}
			key := boltutil.BytesToUint64(k) / 1000000
			result[key] = value.TotalBurnFee
		}
		return nil
	})
	return result, err
}

func (self *BoltStatStorage) GetWalletFee(fromTime uint64, toTime uint64, freq string, reserveAddr ethereum.Address, walletAddr ethereum.Address) (common.StatTicks, error) {
	result := common.StatTicks{}

	err := self.db.Update(func(tx *bolt.Tx) error {
		bucketName := fmt.Sprintf("%s_%s", common.AddrToString(reserveAddr), common.AddrToString(walletAddr))
		b, _ := tx.CreateBucketIfNotExists([]byte(bucketName))
		freqBkName, _ := getBucketNameByFreq(freq)
		freqBk, _ := b.CreateBucketIfNotExists([]byte(freqBkName))

		min := boltutil.Uint64ToBytes(fromTime)
		max := boltutil.Uint64ToBytes(toTime)

		c := freqBk.Cursor()
		for k, v := c.Seek(min); k != nil && bytes.Compare(k, max) <= 0; k, v = c.Next() {
			value := common.BurnFeeStats{}
			if err := json.Unmarshal(v, &value); err != nil {
				return err
			}
			key := boltutil.BytesToUint64(k) / 1000000
			result[key] = value.TotalBurnFee
		}
		return nil
	})

	return result, err
}

func (self *BoltStatStorage) GetUserVolume(fromTime uint64, toTime uint64, freq string, ethUserAddr ethereum.Address) (common.StatTicks, error) {
	result := common.StatTicks{}
	userAddr := common.AddrToString(ethUserAddr)
	err := self.db.Update(func(tx *bolt.Tx) error {
		b, _ := tx.CreateBucketIfNotExists([]byte(userAddr))
		freqBkName, _ := getBucketNameByFreq(freq)
		freqBk, _ := b.CreateBucketIfNotExists([]byte(freqBkName))

		min := boltutil.Uint64ToBytes(fromTime)
		max := boltutil.Uint64ToBytes(toTime)
		c := freqBk.Cursor()
		for k, v := c.Seek(min); k != nil && bytes.Compare(k, max) <= 0; k, v = c.Next() {
			value := common.VolumeStats{}
			if err := json.Unmarshal(v, &value); err != nil {
				return err
			}
			key := boltutil.BytesToUint64(k) / 1000000
			result[key] = value
		}
		return nil
	})
	return result, err
}

func (self *BoltStatStorage) GetReserveVolume(fromTime uint64, toTime uint64, freq string, reserveAddr, token ethereum.Address) (common.StatTicks, error) {
	result := common.StatTicks{}
	err := self.db.Update(func(tx *bolt.Tx) error {
		bucketKey := fmt.Sprintf("%s_%s", common.AddrToString(reserveAddr), common.AddrToString(token))
		b, _ := tx.CreateBucketIfNotExists([]byte(bucketKey))
		freqBkName, _ := getBucketNameByFreq(freq)
		freqBk, _ := b.CreateBucketIfNotExists([]byte(freqBkName))

		min := boltutil.Uint64ToBytes(fromTime)
		max := boltutil.Uint64ToBytes(toTime)
		c := freqBk.Cursor()
		for k, v := c.Seek(min); k != nil && bytes.Compare(k, max) <= 0; k, v = c.Next() {
			value := common.VolumeStats{}
			if err := json.Unmarshal(v, &value); err != nil {
				return err
			}
			key := boltutil.BytesToUint64(k) / 1000000
			result[key] = value
		}
		return nil
	})
	return result, err
}

func (self *BoltStatStorage) GetTokenHeatmap(fromTime, toTime uint64, key, freq string) (common.StatTicks, error) {
	result := common.StatTicks{}
	err := self.db.Update(func(tx *bolt.Tx) error {
		nkey := strings.ToLower(key)
		b, _ := tx.CreateBucketIfNotExists([]byte(nkey))
		freqBkName, _ := getBucketNameByFreq(freq)
		freqBk, _ := b.CreateBucketIfNotExists([]byte(freqBkName))

		min := boltutil.Uint64ToBytes(fromTime)
		max := boltutil.Uint64ToBytes(toTime)
		c := freqBk.Cursor()
		for k, v := c.Seek(min); k != nil && bytes.Compare(k, max) <= 0; k, v = c.Next() {
			value := common.VolumeStats{}
			if err := json.Unmarshal(v, &value); err != nil {
				return err
			}
			key := boltutil.BytesToUint64(k) / 1000000
			result[key] = value
		}
		return nil
	})
	return result, err
}

func (self *BoltStatStorage) SetTradeSummary(tradeSummary map[string]common.MetricStatsTimeZone, lastProcessTimePoint uint64) error {
	var err error
	err = self.db.Update(func(tx *bolt.Tx) error {
		for key, stats := range tradeSummary {
			key = strings.ToLower(key)
			b, _ := tx.CreateBucketIfNotExists([]byte(key))
			// update to timezone buckets
			for i := START_TIMEZONE; i <= END_TIMEZONE; i++ {
				stats := stats[i]
				freq := fmt.Sprintf("%s%d", TIMEZONE_BUCKET_PREFIX, i)
				tzBucket, uErr := b.CreateBucketIfNotExists([]byte(freq))
				if uErr != nil {
					return uErr
				}
				for timepoint, stat := range stats {
					timestamp := boltutil.Uint64ToBytes(timepoint)
					// try get data from this timestamp, if exist then add more data
					currentData := common.MetricStats{}
					v := tzBucket.Get(timestamp)
					if v != nil {
						if uErr := json.Unmarshal(v, &currentData); uErr != nil {
							return uErr
						}
					}
					currentData.ETHVolume += stat.ETHVolume
					currentData.USDVolume += stat.USDVolume
					currentData.BurnFee += stat.BurnFee
					currentData.TradeCount += stat.TradeCount
					currentData.UniqueAddr += stat.UniqueAddr
					currentData.NewUniqueAddresses += stat.NewUniqueAddresses
					currentData.KYCEd += stat.KYCEd
					if currentData.TradeCount > 0 {
						currentData.ETHPerTrade = currentData.ETHVolume / float64(currentData.TradeCount)
						currentData.USDPerTrade = currentData.USDVolume / float64(currentData.TradeCount)
					}
					dataJSON, uErr := json.Marshal(currentData)
					if uErr != nil {
						return uErr
					}
					if uErr = tzBucket.Put(timestamp, dataJSON); uErr != nil {
						return uErr
					}
				}
			}
		}
		lastProcessBk := tx.Bucket([]byte(TRADELOG_PROCESSOR_STATE))
		if lastProcessBk != nil {
			dataJSON := boltutil.Uint64ToBytes(lastProcessTimePoint)
			return lastProcessBk.Put([]byte(TRADE_SUMMARY_AGGREGATION), dataJSON)
		}
		return nil
	})
	return err
}

func (self *BoltStatStorage) GetTradeSummary(fromTime uint64, toTime uint64, timezone int64) (common.StatTicks, error) {
	result := common.StatTicks{}
	tzstring := fmt.Sprintf("%s%d", TIMEZONE_BUCKET_PREFIX, timezone)
	err := self.db.Update(func(tx *bolt.Tx) error {
		b, _ := tx.CreateBucketIfNotExists([]byte("trade_summary"))
		timezoneBk, _ := b.CreateBucketIfNotExists([]byte(tzstring))
		min := boltutil.Uint64ToBytes(fromTime)
		max := boltutil.Uint64ToBytes(toTime)

		c := timezoneBk.Cursor()
		for k, v := c.Seek(min); k != nil && bytes.Compare(k, max) <= 0; k, v = c.Next() {
			summary := common.MetricStats{}
			if err := json.Unmarshal(v, &summary); err != nil {
				return err
			}
			key := boltutil.BytesToUint64(k) / 1000000
			result[key] = summary
		}
		return nil
	})

	return result, err
}
