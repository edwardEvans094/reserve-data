package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"reflect"

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

func (self *HTTPServer) reloadMultipleTokensIndices(tokenListings map[string]common.TokenListing) error {
	tokens, err := self.setting.GetInternalTokens()
	if err != nil {
		return err
	}
	for _, tokenListing := range tokenListings {
		token := tokenListing.Token
		if token.Internal {
			tokens = append(tokens, token)
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

// getCompositeExchangeSetting will query the current exchange setting with key ExName.
// return a struct contain all
func (self *HTTPServer) getCompositeExchangeSetting(exName settings.ExchangeName) (*common.CompositeExchangeSetting, error) {
	exFee, err := self.setting.GetFee(exName)
	if err != nil {
		return nil, err
	}
	exMinDep, err := self.setting.GetMinDeposit(exName)
	if err != nil {
		return nil, err
	}
	exInfos, err := self.setting.GetExchangeInfo(exName)
	if err != nil {
		return nil, err
	}
	depAddrs, err := self.setting.GetDepositAddresses(exName)
	if err != nil {
		return nil, err
	}
	return common.NewCompositeExchangeSetting(depAddrs, exMinDep, exFee, exInfos), nil
}

func (self *HTTPServer) PrepareExchangeSetting(token common.Token, tokExSetts map[string]common.TokenExchangeSetting, preparedExchangeSetting *map[settings.ExchangeName]*common.CompositeExchangeSetting) error {
	for ex, tokExSett := range tokExSetts {
		exName, err := self.EnsureRunningExchange(ex)
		if err != nil {
			return fmt.Errorf("Exchange %s is not in current deployment", ex)
		}
		comExSet, ok := (*preparedExchangeSetting)[exName]
		//create a current compositeExchangeSetting from setting if it does not exist yet
		if !ok {
			comExSet, err = self.getCompositeExchangeSetting(exName)
			if err != nil {
				return err
			}
		}
		//update exchange Deposite Address for compositeExchangeSetting
		comExSet.ExDepositAddress[token.ID] = ethereum.HexToAddress(tokExSett.DepositAddress)

		//update Exchange Info for compositeExchangeSetting
		for pairID, epl := range tokExSett.PrecisionLimit {
			comExSet.ExInfo[pairID] = epl
		}

		//Update Exchange Fee for compositeExchangeSetting
		comExSet.ExFee.Trading[token.ID] = tokExSett.Fee.Trading
		comExSet.ExFee.Funding.Deposit[token.ID] = tokExSett.Fee.Deposit
		comExSet.ExFee.Funding.Withdraw[token.ID] = tokExSett.Fee.WithDraw

		//Update Exchange Min deposit for compositeExchangeSetting
		comExSet.ExMinDeposit[token.ID] = tokExSett.MinDeposit

		(*preparedExchangeSetting)[exName] = comExSet
	}
	return nil
}

func (self *HTTPServer) ConfirmTokenListing(c *gin.Context) {
	postForm, ok := self.Authenticated(c, []string{"data"}, []Permission{ConfigurePermission})
	if !ok {
		return
	}
	data := []byte(postForm.Get("data"))
	var tokenListings map[string]common.TokenListing
	if err := json.Unmarshal(data, &tokenListings); err != nil {
		httputil.ResponseFailure(c, httputil.WithReason(fmt.Sprintf("cant not unmarshall token request %s", err.Error())))
		return
	}
	pdPWI, err := self.metric.GetPendingPWIEquationV2()
	if err != nil {
		log.Printf("WARNING: can not get Pending PWIEquation v2, this will only allow listing non-internal token")
	}
	targetQty, err := self.metric.GetPendingTargetQtyV2()
	if err != nil {
		log.Printf("WARNING: can not get Pending TargetQty v2, this will only allow listing non-internal token")
	}
	pendingTLs, err := self.setting.GetPendingTokenListings()
	if err != nil {
		httputil.ResponseFailure(c, httputil.WithReason(fmt.Sprintf("Can not get pending token listing (%s)", err.Error())))
	}

	preparedExchangeSetting := make(map[settings.ExchangeName]*common.CompositeExchangeSetting)
	preparedToken := []common.Token{}

	for tokenID, tokenListing := range tokenListings {
		token := tokenListing.Token
		preparedToken = append(preparedToken, token)
		//check if there is the token in pending PWIequation
		if _, ok := pdPWI[token.ID]; (token.Internal) && (!ok) {
			httputil.ResponseFailure(c, httputil.WithReason("The Token is not in current pendingPWIEquation "))
			return
		}
		//check if there is the token in pending targetqty
		if _, ok := targetQty[token.ID]; (token.Internal) && (!ok) {
			httputil.ResponseFailure(c, httputil.WithReason("The Token is not in current PendingTargetQty "))
			return
		}

		//check if the token is available in pending token listing and is deep equal to it.
		pendingTL, avail := pendingTLs[tokenID]
		if !avail {
			httputil.ResponseFailure(c, httputil.WithReason("Token %s is not avaiale in pending token listing"))
			return
		}
		if eq := reflect.DeepEqual(pendingTL, tokenListing); !eq {
			httputil.ResponseFailure(c, httputil.WithReason("Confirm and pending token listing request are not equal"))
			return
		}
		if uErr := self.PrepareExchangeSetting(token, tokenListing.Exchange, &preparedExchangeSetting); uErr != nil {
			httputil.ResponseFailure(c, httputil.WithError(uErr))
			return
		}
	}

	//reload token indices if the token is Internal
	if err = self.reloadMultipleTokensIndices(tokenListings); err != nil {
		httputil.ResponseFailure(c, httputil.WithError(err))
		return
	}

	// Apply the change into setting database
	if err = self.setting.ApplyTokenWithExchangeSetting(preparedToken, preparedExchangeSetting); err != nil {
		httputil.ResponseFailure(c, httputil.WithError(err))
		return
	}

	httputil.ResponseSuccess(c)

}

func (self *HTTPServer) RejectTokenListing(c *gin.Context) {
	_, ok := self.Authenticated(c, []string{}, []Permission{ConfirmConfPermission})
	if !ok {
		return
	}
	if err := self.setting.RemovePendingTokenListings(); err != nil {
		httputil.ResponseFailure(c, httputil.WithError(err))
		return
	}
	httputil.ResponseSuccess(c)
}

// ListToken will pre-process the token request and put into pending token request
// It will not apply any change to DB
func (self *HTTPServer) ListToken(c *gin.Context) {
	postForm, ok := self.Authenticated(c, []string{"data"}, []Permission{ConfigurePermission})
	if !ok {
		return
	}
	data := []byte(postForm.Get("data"))
	TokenListings := make(map[string]common.TokenListing)
	if err := json.Unmarshal(data, &TokenListings); err != nil {
		httputil.ResponseFailure(c, httputil.WithReason(fmt.Sprintf("cant not unmarshall token request %s", err.Error())))
		return
	}

	pdPWI, err := self.metric.GetPendingPWIEquationV2()
	if err != nil {
		log.Printf("WARNING: can not get Pending PWIEquation v2, this will only allow listing non-internal token")
	}
	targetQty, err := self.metric.GetPendingTargetQtyV2()
	if err != nil {
		log.Printf("WARNING: can not get Pending TargetQty v2, this will only allow listing non-internal token")
	}
	// prepare each TokenListing instance for individual token
	for tokenID, TokenListing := range TokenListings {
		token := TokenListing.Token
		// To list token, its active status must be true
		if !token.Active {
			httputil.ResponseFailure(c, httputil.WithReason(fmt.Sprintf("The listing token %s is inactive", tokenID)))
			return
		}
		//check if there is the token in pending PWIequation
		if _, ok := pdPWI[token.ID]; (token.Internal) && (!ok) {
			httputil.ResponseFailure(c, httputil.WithReason("The Token is not in current pendingPWIEquation "))
			return
		}

		//check if there is the token in pending targetqty
		if _, ok := targetQty[token.ID]; (token.Internal) && (!ok) {
			httputil.ResponseFailure(c, httputil.WithReason("The Token is not in current PendingTargetQty "))
			return
		}

		//verify exchange status and exchange precision limit for each exchange

		for ex, tokExSett := range TokenListing.Exchange {
			_, uErr := self.EnsureRunningExchange(ex)
			if uErr != nil {
				httputil.ResponseFailure(c, httputil.WithReason(fmt.Sprintf("Exchange %s is not in current deployment", ex)))
				return
			}
			//query exchangeprecisionlimit from exchange for the pair token-ETH
			pairID := common.NewTokenPairID(token.ID, "ETH")

			// If the pair is not in current token listing request, query it from exchange
			_, ok := tokExSett.PrecisionLimit[pairID]
			if !ok {
				exchange, uErr := common.GetExchange(ex)
				if uErr != nil {
					httputil.ResponseFailure(c, httputil.WithReason(fmt.Sprintf("Can not get exchange instance %s (%s)", ex, uErr)))
					return
				}
				epl, uErr := exchange.GetLiveExchangeInfo(pairID)
				if (uErr != nil) && (!ok) {
					httputil.ResponseFailure(c, httputil.WithReason(fmt.Sprintf("Can not prepare exchange info for token on exchange %s (%s). The exchange precision limit of pair %s-ETH must available either on exchange or manually send within the request", ex, uErr, tokenID)))
					return
				}
				if tokExSett.PrecisionLimit == nil {
					tokExSett.PrecisionLimit = make(map[common.TokenPairID]common.ExchangePrecisionLimit)
				}
				tokExSett.PrecisionLimit[pairID] = epl
			}
			TokenListing.Exchange[ex] = tokExSett
		}
		TokenListings[tokenID] = TokenListing
	}
	if err := self.setting.UpdatePendingTokenListings(TokenListings); err != nil {
		httputil.ResponseFailure(c, httputil.WithError(err))
		return
	}
	httputil.ResponseSuccess(c)
	return
}

func (self *HTTPServer) GetPendingTokenListings(c *gin.Context) {
	_, ok := self.Authenticated(c, []string{}, []Permission{RebalancePermission, ConfigurePermission, ReadOnlyPermission, ConfirmConfPermission})
	if !ok {
		return
	}
	data, err := self.setting.GetPendingTokenListings()
	if err != nil {
		httputil.ResponseFailure(c, httputil.WithError(err))
		return
	}
	httputil.ResponseSuccess(c, httputil.WithData(data))
	return
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
	//reload token indices if the token is Internal
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
