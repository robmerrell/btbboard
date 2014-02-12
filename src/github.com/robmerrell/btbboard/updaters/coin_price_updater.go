package updaters

import (
	"encoding/json"
	"github.com/robmerrell/btbboard/models"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"
)

var coinbaseUrl = "https://coinbase.com/api/v1/currencies/exchange_rates"

// the cryptsy API is currenty broken for BTB, so get prices from the full market data.
var cryptsyUrl = "http://pubapi.cryptsy.com/api.php?method=marketdatav2"

type CoinPrice struct{}

// Update retrieves BTB buy prices in both USD and BTC and saves
// the prices to the database.
func (c *CoinPrice) Update() error {
	usd, err := coinbaseQuote()
	if err != nil {
		return err
	}

	cryptsyBtc, err := cryptsyQuote()
	if err != nil {
		return err
	}

	conn := models.CloneConnection()
	defer conn.Close()

	price := &models.Price{
		UsdPerBtc:   usd,
		Cryptsy:     &models.ExchangePrice{Btc: cryptsyBtc, Usd: usd * cryptsyBtc},
		GeneratedAt: time.Now().UTC().Truncate(time.Minute),
	}
	if err := price.SetPercentChange(conn); err != nil {
		return err
	}

	return price.Insert(conn)
}

// coinbaseQuote gets the current USD value for 1 BTC from coinbase.
func coinbaseQuote() (float64, error) {
	resp, err := http.Get(coinbaseUrl)
	if err != nil {
		return 0.0, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	var value struct {
		Btc string `json:"btc_to_usd"`
	}
	if err := json.Unmarshal(body, &value); err != nil {
		return 0.0, err
	}

	return strconv.ParseFloat(value.Btc, 64)
}

// cryptsyQuote gets the current BTB/BTC value from cryptsy.
func cryptsyQuote() (float64, error) {
	resp, err := http.Get(cryptsyUrl)
	if err != nil {
		return 0.0, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	var value struct {
		Return struct {
			Markets map[string]interface{}
		}
	}
	if err := json.Unmarshal(body, &value); err != nil {
		return 0.0, err
	}

	market := value.Return.Markets["BTB/BTC"].(map[string]interface{})
	recentTrades := market["recenttrades"].([]interface{})
	price := recentTrades[0].(map[string]interface{})["price"].(string)

	return strconv.ParseFloat(price, 64)
}
