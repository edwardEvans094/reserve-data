package settings

import (
	"encoding/json"
	"io/ioutil"

	"github.com/KyberNetwork/reserve-data/common"
)

type token struct {
	Address                 string `json:"address"`
	Name                    string `json:"name"`
	Decimal                 int64  `json:"decimals"`
	Active                  bool   `json:"internal use"`
	Internal                bool   `json:"listed"`
	MinimalRecordResolution string `json:"minimalRecordResolution"`
	MaxTotalImbalance       string `json:"maxPerBlockImbalance"`
	MaxPerBlockImbalance    string `json:"maxTotalImbalance"`
}

type TokenConfig struct {
	Tokens map[string]token `json:"tokens"`
}

type TokenSetting struct {
	Storage TokenStorage
}

func NewTokenSetting(tokenStorage TokenStorage) (*TokenSetting, error) {
	tokenSetting := TokenSetting{tokenStorage}
	return &tokenSetting, nil

}
func (setting *Settings) loadTokenFromFile(filePath string) error {
	data, err := ioutil.ReadFile(filePath)
	tokens := TokenConfig{}
	if err != nil {
		return err
	}
	if err = json.Unmarshal(data, &tokens); err != nil {
		return err
	}
	for id, t := range tokens.Tokens {
		token := common.NewToken(id, t.Name, t.Address, t.Decimal, t.Active, t.Internal, t.MinimalRecordResolution, t.MaxPerBlockImbalance, t.MaxTotalImbalance)
		if err = setting.UpdateToken(token); err != nil {
			return err
		}
	}
	return nil
}
