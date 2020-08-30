package stonks

import (
	"context"
	"errors"
	"fmt"
	finnhub "github.com/Finnhub-Stock-API/finnhub-go"
	"github.com/antihax/optional"
	//	"github.com/go-chat-bot/bot"
	"log"
	"os"
	"strings"
)

func GetPreRonaPrice(finnhubClient *finnhub.DefaultApiService, auth context.Context, symbol string) (price float32) {
	var ronaSeconds int64
	ronaSeconds = 1580882400 // feb 5
	var ronaNextDaySeconds int64
	ronaNextDaySeconds = ronaSeconds + (10 * (60 * 60 * 24)) // feb 15
	//fmt.Println(ronaNextDaySeconds)
	stockCandles, _, err := finnhubClient.StockCandles(auth, symbol, "D", ronaSeconds, ronaNextDaySeconds, nil)
	if err != nil {
		log.Fatal(err)
	}
	//fmt.Printf("%+v\n", stockCandles)
	//fmt.Printf("day1 close %5.2f, day10 close %5.2f\n", stockCandles.C[0], stockCandles.C[len(stockCandles.C)-1])
	price = (stockCandles.C[0] + stockCandles.C[len(stockCandles.C)-1]) / 2
	return price

}
func GetDailyChange(quote finnhub.Quote) (percent float32) {
	percent = (quote.C - quote.Pc) / quote.Pc
	return percent
}

func Quote(symbol string, preRona bool) (msg string, err error) {
	finnhubClient := finnhub.NewAPIClient(finnhub.NewConfiguration()).DefaultApi
	auth := context.WithValue(context.Background(), finnhub.ContextAPIKey, finnhub.APIKey{
		Key: os.Getenv("FINNHUB_API_KEY"),
	})

	log.Printf("Looking up stock quote: %s\n", symbol)
	quote, _, err := finnhubClient.Quote(auth, symbol)
	if err != nil {
		msg = "error?"
		return msg, err
	}
	if quote.Pc == 0 && quote.O == 0 {
		msg = fmt.Sprintf("No data found for symbol %s\n", symbol)
		return msg, errors.New(msg)
	}

	log.Printf("%+v\n", quote)
	var preRonaPrice float32
	if preRona {
		preRonaPrice = GetPreRonaPrice(finnhubClient, auth, symbol)
	}
	//profile, _, err := finnhubClient.CompanyProfile2(auth, &finnhub.CompanyProfile2Opts{Symbol: optional.NewString(symbol)})
	dailyChange := GetDailyChange(quote)
	//log.Printf("%+v\n", profile)
	if preRona {
		msg = fmt.Sprintf("[%s] Price: %5.2f  // Today: %5.2f%% PreRonaPrice: %5.2f", symbol, quote.C, dailyChange, preRonaPrice)
	} else {
		msg = fmt.Sprintf("[%s] Price: %5.2f  // Today: %5.2f%%", symbol, quote.C, dailyChange)
	}
	log.Printf("%+v\n", msg)

	return msg, nil
}

func CompanyProfile(sym string) (msg string, err error) {
	// Company profile2
	symbol := strings.ToUpper(sym)
	finnhubClient := finnhub.NewAPIClient(finnhub.NewConfiguration()).DefaultApi
	auth := context.WithValue(context.Background(), finnhub.ContextAPIKey, finnhub.APIKey{
		Key: os.Getenv("FINNHUB_API_KEY"),
	})

	profile2, _, err := finnhubClient.CompanyProfile2(auth, &finnhub.CompanyProfile2Opts{Symbol: optional.NewString(symbol)})
	fmt.Printf("%+v\n", profile2)

	return "yeet", nil
}
