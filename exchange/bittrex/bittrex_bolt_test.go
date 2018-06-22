package bittrex

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/KyberNetwork/reserve-data/common"
)

func TestBittrexStorage(t *testing.T) {
	boltFile := "test_bittrex_storage.db"
	tmpDir, err := ioutil.TempDir("", "bittrex_storage")
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
		t.Fatalf("Could not init bittrex bolt storage: %s", err.Error())
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
		t.Fatal("Bittrex get wrong trade history")
	}

}
