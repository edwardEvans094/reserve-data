package blockchain

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"
)

const gasStationURL = "https://ethgasstation.info/json/ethgasAPI.json"

type gasStationResponse struct {
	Standard float64 `json:"average"`
	Fast     float64 `json:"fast"`
	SafeLow  float64 `json:"safeLow"`
}

// GasOracle is an ETH Gas Station client.
type GasOracle struct {
	client *http.Client
	m      sync.RWMutex

	standard float64
	fast     float64
	safeLow  float64
}

// NewGasOracle create new GasOracle instance.
func NewGasOracle() *GasOracle {
	client := &http.Client{Timeout: 10 * time.Second}
	return &GasOracle{client: client}
}

// Standard returns standard gas pricing.
func (gso *GasOracle) Standard() float64 {
	gso.m.RLock()
	defer gso.m.RLock()

	return gso.standard
}

// Fast returns the fast gas pricing.
func (gso *GasOracle) Fast() float64 {
	gso.m.RLock()
	defer gso.m.RLock()

	return gso.fast
}

// SafeLow returns the safe low gas pricing.
func (gso *GasOracle) SafeLow() float64 {
	gso.m.RLock()
	defer gso.m.RLock()

	return gso.safeLow
}

// Fetch periodically polls ETH Gas Station server and update the
// recommendation gas prices.
func (gso *GasOracle) Fetch() {
	ticker := time.NewTicker(10 * time.Second)
	go func() {
		for {
			err := gso.runGasPricing()
			if err != nil {
				log.Printf("Error pricing gas from Gasstation: %v", err)
			}
			<-ticker.C
		}
	}()
}

func (gso *GasOracle) runGasPricing() error {
	gso.m.Lock()
	defer gso.m.Unlock()

	r, err := gso.client.Get(gasStationURL)
	if err != nil {
		return err
	}
	defer func() {
		if cErr := r.Body.Close(); cErr != nil {
			log.Printf("Response close error: %s", cErr.Error())
		}
	}()

	rsp := &gasStationResponse{}
	if err = json.NewDecoder(r.Body).Decode(rsp); err != nil {
		return err
	}

	gso.standard = rsp.Standard
	gso.safeLow = rsp.SafeLow
	gso.fast = rsp.Fast
	return nil
}
