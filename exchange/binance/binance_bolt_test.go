package binance

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/KyberNetwork/reserve-data/common"
)

func TestBinanceStorage(t *testing.T) {
	boltFile := "test_binance_bolt.db"
	tmpDir, err := ioutil.TempDir("", "binance_storage")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if rErr := os.RemoveAll(tmpDir); rErr != nil {
			t.Error(rErr)
		}
	}()

	exchangeTradeHistory := common.ExchangeTradeHistory{
		common.TokenPairID("OMGETH"): []common.TradeHistory{
			{
				ID:        "12342",
				Price:     0.132131,
				Qty:       12.3123,
				Type:      "buy",
				Timestamp: 1528949872,
			},
		},
	}

	storage, err := NewBoltStorage(filepath.Join(tmpDir, boltFile))
	if err != nil {
		t.Fatalf("Could not init binance bolt storage: %s", err.Error())
	}

	// store trade history
	err = storage.StoreTradeHistory(exchangeTradeHistory)
	if err != nil {
		t.Fatal(err)
	}

	// get trade history
	var tradeHistory common.ExchangeTradeHistory
	fromTime := uint64(1528934400000)
	toTime := uint64(1529020800000)
	tradeHistory, err = storage.GetTradeHistory(fromTime, toTime)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(tradeHistory, exchangeTradeHistory) {
		t.Fatal("Get wrong trade history")
	}

	// get last trade history id
	var lastHistoryID string
	lastHistoryID, err = storage.GetLastIDTradeHistory("OMGETH")
	if err != nil {
		t.Fatalf("Get last trade history id error: %s", err.Error())
	}
	if lastHistoryID != "12342" {
		t.Fatalf("Get last trade history wrong")
	}
}
