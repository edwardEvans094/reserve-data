package common

import ethereum "github.com/ethereum/go-ethereum/common"

// TokenExchangeSetting contains necessary information on exchange to List a token on the fly
type TokenExchangeSetting struct {
	DepositAddress string
	Info           ExchangeInfo
	Fee            TokenFee
	MinDeposit     float64
}

type TokenListing struct {
	Token    Token
	Exchange map[string]TokenExchangeSetting
}

type TokenFee struct {
	Trading  float64
	WithDraw float64
	Deposit  float64
}

type Token struct {
	ID                      string `json:"id"`
	Name                    string `json:"name"`
	Address                 string `json:"address"`
	Decimal                 int64  `json:"decimals"`
	Active                  bool   `json:"active"`
	Internal                bool   `json:"internal"`
	MinimalRecordResolution string `json:"minimal_record_resolution"`
	MaxTotalImbalance       string `json:"max_total_imbalance"`
	MaxPerBlockImbalance    string `json:"max_per_block_imbalance"`
}

// NewToken creates a new Token.
func NewToken(id, name, address string, decimal int64, active, internal bool, miminalrr, maxti, maxpbi string) Token {
	return Token{
		ID:                      id,
		Address:                 address,
		Decimal:                 decimal,
		Active:                  active,
		Internal:                internal,
		MinimalRecordResolution: miminalrr,
		MaxTotalImbalance:       maxti,
		MaxPerBlockImbalance:    maxpbi,
	}
}

func (self Token) IsETH() bool {
	return self.ID == "ETH"
}

type TokenPair struct {
	Base  Token
	Quote Token
}

func NewTokenPair(base, quote Token) TokenPair {
	return TokenPair{base, quote}
}

func (self *TokenPair) PairID() TokenPairID {
	return NewTokenPairID(self.Base.ID, self.Quote.ID)
}

func GetTokenAddressesList(tokens []Token) []ethereum.Address {
	tokenAddrs := []ethereum.Address{}
	for _, tok := range tokens {
		if tok.ID != "ETH" {
			tokenAddrs = append(tokenAddrs, ethereum.HexToAddress(tok.Address))
		}
	}
	return tokenAddrs
}
