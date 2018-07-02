package cmd

import (
	"log"
	"strconv"
	"time"

	"github.com/KyberNetwork/reserve-data/cmd/comparerates"
	"github.com/KyberNetwork/reserve-data/common"
	"github.com/spf13/cobra"
)

const (
	sleepTime                  time.Duration = 60 //sleep time for forever run mode
	defaultCompareRatesBaseURL               = "https://internal-mainnet-core.kyber.network"
)

var baseURL string
var fromTime string
var toTime string

func compareratestart(cmd *cobra.Command, args []string) {
	kyberENV := common.RunningMode()
	params := make(map[string]string)
	params["fromTime"] = fromTime
	params["toTime"] = toTime
	config := GetConfigFromENV(kyberENV)
	if len(params["toTime"]) < 1 {
		log.Printf("There was no end time, go to foverer run mode...")
		for {
			params["toTime"] = strconv.FormatInt((time.Now().UnixNano() / int64(time.Millisecond)), 10)
			comparerates.DoQuery(baseURL, params, *config)
			time.Sleep(sleepTime * time.Second)
			params["fromTime"] = params["toTime"]
		}

	} else {
		log.Printf("Go to single query returning mode")
		comparerates.DoQuery(baseURL, params, *config)
	}

}

var compareRates = &cobra.Command{
	Use:   "compare ",
	Short: "compare rate from get_all_rates to setRate activities",
	Long:  `call get_all_rates and get_activities API to server and compare the rates between set_rate activites and get_rate activites from the same block, alert if the rate differ is >0.1%`,
	Run:   compareratestart,
}

func init() {
	//compare rate flags
	compareRates.Flags().StringVar(&baseURL, "url", defaultCompareRatesBaseURL, "base URL for API query")
	compareRates.Flags().StringVar(&fromTime, "from_time", "", "begining time for query, required params")
	if err := compareRates.MarkFlagRequired("from_time"); err != nil {
		log.Fatalf(err.Error())
	}
	compareRates.Flags().StringVar(&toTime, "to_time", "", "end time of querying, if not set then the program will run until force quit")
	RootCmd.AddCommand(compareRates)
}
