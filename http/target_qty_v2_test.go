package http

import (
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/KyberNetwork/reserve-data/core"
	"github.com/KyberNetwork/reserve-data/data"
	"github.com/KyberNetwork/reserve-data/data/storage"
	"github.com/KyberNetwork/reserve-data/http/httputil"
	ethereum "github.com/ethereum/go-ethereum/common"
	"github.com/gin-gonic/gin"
)

//TestHTTPServerTargetQtyV2 check if api v2 work correctly
func TestHTTPServerTargetQtyV2(t *testing.T) {
	const (
		storePendingTargetQtyV2 = "/v2/settargetqty"
		getPendingTargetQtyV2   = "/v2/pendingtargetqty"
		confirmTargetQtyV2      = "/v2/confirmtargetqty"
		rejectTargetQtyV2       = "/v2/canceltargetqty"
		getTargetQtyV2          = "/v2/targetqty"
		testData                = `{
  "EOS": {
      "total_target": 750,
      "reserve_target": 500,
	  "rebalance_threshold": 0.25,
	  "transfer_threshold": 0.343
  },
  "ETH": {
      "total_target": 750,
      "reserve_target": 500,
	  "rebalance_target": 0.25,
	  "transfer_threshold": 0.343,
	  "minimum_amount" : {
		"huobi" : 10,
		"binance" : 20
	  }, "exchange_ratio":  {
		"huobi": 3,
		"bianace": 4
	}
  }
}
`
		testDataWrongConfirmation = `{
  "EOS": {
      "total_target": 751,
      "reserve_target": 500,
	  "rebalance_threshold": 0.25,
	  "transfer_threshold": 0.343
    },
  "ETH": {
      "total_target": 750,
      "reserve_target": 500,
	  "rebalance_target": 0.25,
	  "transfer_threshold": 0.343,
	  "minimum_amount" : {
		"huobi" : 10,
		"binance" : 20
	  },
	  "exchange_ratio":  {
		"huobi": 3,
		"bianace": 4
	}
  }
}
`
	)
	tmpDir, err := ioutil.TempDir("", "test_target_qty_v2")
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		if rErr := os.RemoveAll(tmpDir); rErr != nil {
			t.Error(rErr)
		}
	}()

	st, err := storage.NewBoltStorage(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}

	s := HTTPServer{
		app:         data.NewReserveData(st, nil, nil, nil, nil, nil),
		core:        core.NewReserveCore(nil, st, ethereum.Address{}),
		metric:      st,
		authEnabled: false,
		r:           gin.Default()}
	s.register()

	var tests = []testCase{
		{
			msg:      "getting non exists pending target quantity",
			endpoint: getPendingTargetQtyV2,
			method:   http.MethodGet,
			assert:   httputil.ExpectFailure,
		},
		{
			msg:      "getting non exists target",
			endpoint: getTargetQtyV2,
			method:   http.MethodGet,
			assert:   httputil.ExpectFailure,
		},
		{
			msg:      "confirm when no pending target quantity request exists",
			endpoint: confirmTargetQtyV2,
			method:   http.MethodPost,
			data: map[string]string{
				"value": testData,
			},
			assert: httputil.ExpectFailure,
		},
		{
			msg:      "reject when no pending target quantity request exists",
			endpoint: rejectTargetQtyV2,
			method:   http.MethodPost,
			assert:   httputil.ExpectFailure,
		},
		{
			msg:      "valid post form",
			endpoint: storePendingTargetQtyV2,
			method:   http.MethodPost,
			data: map[string]string{
				"value": testData,
			},
			assert: httputil.ExpectSuccess,
		},
		{
			msg:      "setting when pending exists",
			endpoint: storePendingTargetQtyV2,
			method:   http.MethodPost,
			data: map[string]string{
				"value": testData,
			},
			assert: httputil.ExpectFailure,
		},
		{
			msg:      "confirm with wrong data",
			endpoint: confirmTargetQtyV2,
			method:   http.MethodPost,
			data: map[string]string{
				"value": testDataWrongConfirmation,
			},
			assert: httputil.ExpectFailure,
		},
		{
			msg:      "confirm with correct data",
			endpoint: confirmTargetQtyV2,
			method:   http.MethodPost,
			data: map[string]string{
				"value": testData,
			},
			assert: httputil.ExpectSuccess,
		},
		{
			msg:      "valid post form",
			endpoint: storePendingTargetQtyV2,
			method:   http.MethodPost,
			data: map[string]string{
				"value": testData,
			},
			assert: httputil.ExpectSuccess,
		},
		{
			msg:      "reject when there is pending equation",
			endpoint: rejectTargetQtyV2,
			method:   http.MethodPost,
			data: map[string]string{
				"value": "some random post form or this request will be unauthenticated",
			},
			assert: httputil.ExpectSuccess,
		},
	}

	for _, tc := range tests {
		t.Run(tc.msg, func(t *testing.T) { testHTTPRequest(t, tc, s.r) })
	}
}
