package http

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/KyberNetwork/reserve-data/common"
	"github.com/KyberNetwork/reserve-data/http/httputil"
	"github.com/KyberNetwork/reserve-data/settings"
	ethereum "github.com/ethereum/go-ethereum/common"
	"github.com/gin-gonic/gin"
)

func removeTokenFromList(tokens []common.Token, t common.Token) ([]common.Token, error) {
	if len(tokens) == 0 {
		return tokens, errors.New("Internal Token list is empty")
	}
	for i, token := range tokens {
		if token.ID == t.ID {
			tokens[len(tokens)-1], tokens[i] = tokens[i], tokens[len(tokens)-1]
			return tokens[:len(tokens)-1], nil
		}
	}
	return tokens, fmt.Errorf("The deactivating token %s is not in current internal token list", t.ID)
}

func (self *HTTPServer) reloadTokenIndices(newToken common.Token, active bool) error {
	tokens, err := self.setting.GetInternalTokens()
	if err != nil {
		return err
	}
	if active {
		tokens = append(tokens, newToken)
	} else {
		if tokens, err = removeTokenFromList(tokens, newToken); err != nil {
			return err
		}
	}
	if err = self.blockchain.LoadAndSetTokenIndices(common.GetTokenAddressesList(tokens)); err != nil {
		return err
	}
	return nil
}

func (self *HTTPServer) EnsureRunningExchange(ex string) (settings.ExchangeName, error) {
	exName, ok := settings.ExchangTypeValues()[ex]
	if !ok {
		return exName, fmt.Errorf("Exchange %s is not in current deployment", ex)
	}
	exStatuses, err := self.setting.GetExchangeStatus()
	if err != nil {
		return exName, fmt.Errorf("Can not get Exchange status %s", err.Error())
	}
	status, ok := exStatuses[ex]
	if !ok {
		return exName, fmt.Errorf("Exchange %s is not in current running exchange", ex)
	}
	if !status.Status {
		return exName, fmt.Errorf("Exchange %s is not currently active", ex)
	}
	return exName, nil
}

func (self *HTTPServer) PrepareExchangeDepositAddress(tokenID, depAddrStr string, exName settings.ExchangeName) (common.ExchangeAddresses, error) {
	depAddr := ethereum.HexToAddress(depAddrStr)
	depAddrs, err := self.setting.GetDepositAddresses(exName)
	if err != nil {
		return depAddrs, err
	}
	depAddrs[tokenID] = depAddr
	return depAddrs, nil
}

func (self *HTTPServer) PrepareExchangeInfo(epls map[common.TokenPairID]common.ExchangePrecisionLimit, exName settings.ExchangeName) (common.ExchangeInfo, error) {
	exInfos, err := self.setting.GetExchangeInfo(exName)
	if err != nil {
		return exInfos, err
	}
	for tokenPairID, epl := range epls {
		exInfos[tokenPairID] = epl
	}
	return exInfos, nil
}

func (self *HTTPServer) PrepareExchangeFees(tokenID string, tokenFee common.TokenFee, exName settings.ExchangeName) (common.ExchangeFees, error) {
	exFee, err := self.setting.GetFee(exName)
	if err != nil {
		return exFee, err
	}
	exFee.Trading[tokenID] = tokenFee.Trading
	exFee.Funding.Deposit[tokenID] = tokenFee.Deposit
	exFee.Funding.Withdraw[tokenID] = tokenFee.WithDraw
	return exFee, err
}

func (self *HTTPServer) PrepareExchangeMinDeposit(tokenID string, minDeposit float64, exName settings.ExchangeName) (common.ExchangesMinDeposit, error) {
	exMinDep, err := self.setting.GetMinDeposit(exName)
	if err != nil {
		return exMinDep, err
	}
	exMinDep[tokenID] = minDeposit
	return exMinDep, nil
}

func (self *HTTPServer) ListToken(c *gin.Context) {
	postForm, ok := self.Authenticated(c, []string{"data"}, []Permission{RebalancePermission, ConfigurePermission})
	if !ok {
		return
	}
	data := []byte(postForm.Get("data"))
	var tokenRequest common.TokenRequest
	if err := json.Unmarshal(data, &tokenRequest); err != nil {
		httputil.ResponseFailure(c, httputil.WithError(err))
		return
	}
	token := tokenRequest.Token
	//reload token indices if the token is Internal
	if token.Internal {
		if err := self.reloadTokenIndices(token, token.Internal); err != nil {
			httputil.ResponseFailure(c, httputil.WithError(err))
			return
		}
	}
	//prepare all the exchange setting.
	preparedExchangeSetting := make(map[settings.ExchangeName]*common.CompositeExchangeSetting)
	for ex, exTokenSetting := range tokenRequest.Exchange {
		exName, uErr := self.EnsureRunningExchange(ex)
		if uErr != nil {
			httputil.ResponseFailure(c, httputil.WithReason(fmt.Sprintf("Exchange %s is not in current deployment", ex)))
			return
		}

		exAddresses, uErr := self.PrepareExchangeDepositAddress(token.ID, exTokenSetting.DepositAddress, exName)
		if uErr != nil {
			httputil.ResponseFailure(c, httputil.WithReason(fmt.Sprintf("Can not prepare exchange address for token on exchange %s (%s)", ex, uErr)))
			return
		}
		exInfo, uErr := self.PrepareExchangeInfo(exTokenSetting.PrecisionLimit, exName)
		if uErr != nil {
			httputil.ResponseFailure(c, httputil.WithReason(fmt.Sprintf("Can not prepare exchange info for token on exchange %s (%s)", ex, uErr)))
			return
		}
		exFees, uErr := self.PrepareExchangeFees(token.ID, exTokenSetting.Fee, exName)
		if uErr != nil {
			httputil.ResponseFailure(c, httputil.WithReason(fmt.Sprintf("Can not prepare exchange fee for token on exchange %s (%s)", ex, uErr)))
			return
		}
		exMinDeposit, uErr := self.PrepareExchangeMinDeposit(token.ID, exTokenSetting.MinDeposit, exName)
		if uErr != nil {
			httputil.ResponseFailure(c, httputil.WithReason(fmt.Sprintf("Can not prepare exchange min deposit for token on exchange %s (%s)", ex, uErr)))
			return
		}
		preparedExchangeSetting[exName] = common.NewCompositeExchangeSetting(exAddresses, exMinDeposit, exFees, exInfo)
	}
	pdPWI, err := self.metric.GetPendingPWIEquationV2()
	if err != nil {
		httputil.ResponseFailure(c, httputil.WithReason(fmt.Sprintf("Can not get pendingPWIEquation %s", err)))
		return
	}
	if _, ok := pdPWI[token.ID]; !ok {
		httputil.ResponseFailure(c, httputil.WithReason("The Token is not in current pending PWIEquation "))
		return
	}
	if err := self.setting.UpdateTokenWithExchangeSetting(token, preparedExchangeSetting); err != nil {
		httputil.ResponseFailure(c, httputil.WithError(err))
	}
	httputil.ResponseSuccess(c)
}

// UpdateToken update minor independent details about a token
// It provides the most simple way to modify  token's information without affecting other component
// To list/delist token, use ListToken/ DelistToken API instead.
func (self *HTTPServer) UpdateToken(c *gin.Context) {
	postForm, ok := self.Authenticated(c, []string{"data"}, []Permission{RebalancePermission, ConfigurePermission})
	if !ok {
		return
	}
	data := []byte(postForm.Get("data"))
	var token common.Token
	if err := json.Unmarshal(data, &token); err != nil {
		httputil.ResponseFailure(c, httputil.WithError(err))
		return
	}
	if _, err := self.setting.GetTokenByID(token.ID); err != nil {
		httputil.ResponseFailure(c, httputil.WithError(err))
		return
	}
	//We only concern reloading indices if the token is Internal
	if token.Internal {
		if err := self.reloadTokenIndices(token, token.Internal); err != nil {
			httputil.ResponseFailure(c, httputil.WithError(err))
			return
		}
	}

	if err := self.setting.UpdateToken(token); err != nil {
		httputil.ResponseFailure(c, httputil.WithError(err))
		return
	}
	httputil.ResponseSuccess(c)
}

func (self *HTTPServer) TokenSettings(c *gin.Context) {
	_, ok := self.Authenticated(c, []string{}, []Permission{RebalancePermission, ConfigurePermission, ReadOnlyPermission, ConfirmConfPermission})
	if !ok {
		return
	}
	data, err := self.setting.GetAllTokens()
	if err != nil {
		httputil.ResponseFailure(c, httputil.WithError(err))
		return
	}
	httputil.ResponseSuccess(c, httputil.WithData(data))
	return
}

func (self *HTTPServer) UpdateAddress(c *gin.Context) {
	postForm, ok := self.Authenticated(c, []string{"name", "address"}, []Permission{RebalancePermission, ConfigurePermission})
	if !ok {
		return
	}
	addrStr := postForm.Get("address")
	name := postForm.Get("name")
	addressName, ok := settings.AddressNameValues()[name]
	if !ok {
		httputil.ResponseFailure(c, httputil.WithReason(fmt.Sprintf("invalid address name: %s", name)))
		return
	}
	addr := ethereum.HexToAddress(addrStr)
	if err := self.setting.UpdateAddress(addressName, addr); err != nil {
		httputil.ResponseFailure(c, httputil.WithError(err))
	}
	httputil.ResponseSuccess(c)
}

func (self *HTTPServer) AddAddressToSet(c *gin.Context) {
	postForm, ok := self.Authenticated(c, []string{"setname", "address"}, []Permission{RebalancePermission, ConfigurePermission})
	if !ok {
		return
	}
	addrStr := postForm.Get("address")
	addr := ethereum.HexToAddress(addrStr)
	setName := postForm.Get("setname")
	addrSetName, ok := settings.AddressSetNameValues()[setName]
	if !ok {
		httputil.ResponseFailure(c, httputil.WithReason(fmt.Sprintf("invalid address set name: %s", setName)))
		return
	}
	if err := self.setting.AddAddressToSet(addrSetName, addr); err != nil {
		httputil.ResponseFailure(c, httputil.WithError(err))
	}
	httputil.ResponseSuccess(c)
}

func (self *HTTPServer) UpdateExchangeFee(c *gin.Context) {
	postForm, ok := self.Authenticated(c, []string{"name", "data"}, []Permission{RebalancePermission, ConfigurePermission})
	if !ok {
		return
	}
	name := postForm.Get("name")
	exName, ok := settings.ExchangTypeValues()[name]
	if !ok {
		httputil.ResponseFailure(c, httputil.WithError(fmt.Errorf("Exchange %s is not in current deployment", name)))
		return
	}
	data := []byte(postForm.Get("data"))
	var exFee common.ExchangeFees
	if err := json.Unmarshal(data, &exFee); err != nil {
		httputil.ResponseFailure(c, httputil.WithError(err))
		return
	}
	if err := self.setting.UpdateFee(exName, exFee); err != nil {
		httputil.ResponseFailure(c, httputil.WithError(err))
	}
	httputil.ResponseSuccess(c)
}

func (self *HTTPServer) UpdateExchangeMinDeposit(c *gin.Context) {
	postForm, ok := self.Authenticated(c, []string{"name", "data"}, []Permission{RebalancePermission, ConfigurePermission})
	if !ok {
		return
	}
	name := postForm.Get("name")
	exName, ok := settings.ExchangTypeValues()[name]
	if !ok {
		httputil.ResponseFailure(c, httputil.WithReason(fmt.Sprintf("Exchange %s is not in current deployment", name)))
		return
	}
	data := []byte(postForm.Get("data"))
	var exMinDeposit common.ExchangesMinDeposit
	if err := json.Unmarshal(data, &exMinDeposit); err != nil {
		httputil.ResponseFailure(c, httputil.WithError(err))
		return
	}
	if err := self.setting.UpdateMinDeposit(exName, exMinDeposit); err != nil {
		httputil.ResponseFailure(c, httputil.WithError(err))
	}
	httputil.ResponseSuccess(c)
}

func (self *HTTPServer) UpdateDepositAddress(c *gin.Context) {
	postForm, ok := self.Authenticated(c, []string{"name", "data"}, []Permission{RebalancePermission, ConfigurePermission})
	if !ok {
		return
	}
	name := postForm.Get("name")
	exName, ok := settings.ExchangTypeValues()[name]
	if !ok {
		httputil.ResponseFailure(c, httputil.WithReason(fmt.Sprintf("Exchange %s is not in current deployment", name)))
		return
	}
	data := []byte(postForm.Get("data"))
	var exDepositAddress common.ExchangeAddresses
	if err := json.Unmarshal(data, &exDepositAddress); err != nil {
		httputil.ResponseFailure(c, httputil.WithError(err))
		return
	}
	if err := self.setting.UpdateDepositAddress(exName, exDepositAddress); err != nil {
		httputil.ResponseFailure(c, httputil.WithError(err))
	}
	httputil.ResponseSuccess(c)
}

func (self *HTTPServer) UpdateExchangeInfo(c *gin.Context) {
	postForm, ok := self.Authenticated(c, []string{"name", "data"}, []Permission{RebalancePermission, ConfigurePermission})
	if !ok {
		return
	}
	name := postForm.Get("name")
	exName, ok := settings.ExchangTypeValues()[name]
	if !ok {
		httputil.ResponseFailure(c, httputil.WithReason(fmt.Sprintf("Exchange %s is not in current deployment", name)))
		return
	}
	data := []byte(postForm.Get("data"))
	var exInfo common.ExchangeInfo
	if err := json.Unmarshal(data, &exInfo); err != nil {
		httputil.ResponseFailure(c, httputil.WithError(err))
		return
	}
	if err := self.setting.UpdateExchangeInfo(exName, exInfo); err != nil {
		httputil.ResponseFailure(c, httputil.WithError(err))
	}
	httputil.ResponseSuccess(c)
}
