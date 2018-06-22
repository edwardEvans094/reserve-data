package common

import ethereum "github.com/ethereum/go-ethereum/common"

type Token struct {
	ID                      string
	Name                    string
	Address                 string
	Decimal                 int64
	Active                  bool
	Internal                bool
	MinimalRecordResolution string
	MaxTotalImbalance       string
	MaxPerBlockImbalance    string
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
