package stonks

import (
	"context"
	"fmt"
	finnhub "github.com/Finnhub-Stock-API/finnhub-go"
	"github.com/antihax/optional"
	"io/ioutil"
	"encoding/csv"
	"errors"
	"log"
	"os"
	"strings"
)

type QuoteDetail struct {
	Symbol             string
	Price              float32
	HighPrice          float32
	LowPrice           float32
	OpenPrice          float32
	PreviousClosePrice float32
	DailyChange        float32
	PreRonaPrice       float32
	Description        string
	FormattedDetail    string
}

type StonksClient struct {
	Fh      *finnhub.DefaultApiService
	Fhauth  context.Context
	Records [][]string
}

func NewStonksClient(finnhubApiKey string, stonksDataPath string) *StonksClient {
	finnhubClient := finnhub.NewAPIClient(finnhub.NewConfiguration()).DefaultApi
	finnhubAuth := context.WithValue(context.Background(), finnhub.ContextAPIKey, finnhub.APIKey{
		Key: finnhubApiKey,
	})
	records, err := GetStonksDataFromCSV(stonksDataPath)
	if err != nil {
		log.Fatal(err)
	}
	client := StonksClient{
		Fh:      finnhubClient,
		Fhauth:  finnhubAuth,
		Records: records,
	}
	return &client
}

func (s *StonksClient) ZQuote(symbol string) (float32, error) {
	log.Printf("Looking up stock quote: %s\n", symbol)
	quote, _, err := s.Fh.Quote(s.Fhauth, symbol)
	if err != nil {
		return 0.0, err
	}
	if quote.Pc == 0 && quote.O == 0 {
		msg := fmt.Sprintf("No data found for symbol %s\n", symbol)
		//log.Printf(msg)
		return 0, errors.New(msg)
	}

	return quote.C, nil
}

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
	if len(stockCandles.C) == 0 {
		log.Printf("%s is newer than rona, no PreRonaPrice calculated\n", symbol)
		return 0.0
	}
	price = (stockCandles.C[0] + stockCandles.C[len(stockCandles.C)-1]) / 2
	return price

}
func GetDailyChange(quote finnhub.Quote) (percent float32) {
	percent = 100 * ((quote.C - quote.Pc) / quote.Pc)
	return percent
}

func (s *StonksClient) Quote(symbol string) (detail QuoteDetail, err error) {
	log.Printf("Looking up stock quote: %s\n", symbol)
	quote, _, err := s.Fh.Quote(s.Fhauth, symbol)
	if err != nil {
		detail = QuoteDetail{FormattedDetail: "error?"}
		return detail, err
	}
	if quote.Pc == 0 && quote.O == 0 {
		msg := fmt.Sprintf("No data found for symbol %s\n", symbol)
		detail = QuoteDetail{FormattedDetail: msg}
		return detail, errors.New(msg)
	}

	log.Printf("%+v\n", quote)
	var preRonaPrice float32
	preRonaPrice = GetPreRonaPrice(s.Fh, s.Fhauth, symbol)

	desc, err := GetStonkDescription(s.Records, symbol)
	dailyChange := GetDailyChange(quote)
	//log.Printf("%+v\n", profile)
	var msg string
	msg = fmt.Sprintf("```\n [%s] %s \n Price: %5.2f \n Today: %5.2f%% PreRonaPrice: %5.2f```", symbol, desc, quote.C, dailyChange, preRonaPrice)
	log.Printf("%+v\n", msg)
	detail = QuoteDetail{
		Symbol:             symbol,
		Price:              quote.C,
		HighPrice:          quote.H,
		LowPrice:           quote.L,
		OpenPrice:          quote.O,
		PreviousClosePrice: quote.Pc,
		DailyChange:        dailyChange,
		PreRonaPrice:       preRonaPrice,
		Description:        desc,
		FormattedDetail:    msg,
	}

	return detail, nil
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

func GetStonkDescription(records [][]string, symbol string) (string, error) {
	upcased := strings.ToUpper(symbol)
	fmt.Println("Searching for", upcased)
	for _, record := range records {
		if record[0] == upcased {
			description := record[1]
			return description, nil
		}
	}

	return "", errors.New("symbol not found")
}

func GetStonksDataFromCSV(path string) ([][]string, error) {

	// stonksdata.txt from:
	// ftp://ftp.nasdaqtrader.com/SymbolDirectory/
	// cat nasdaqlisted.txt otherlisted.txt mfundslist.txt |cut -d "|" -f 1-3 > stonksdata.txt
	dat, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}

	//fmt.Print(string(dat))
	r := csv.NewReader(strings.NewReader(string(dat)))
	r.Comma = '|'

	records, err := r.ReadAll()
	if err != nil {
		return nil, err
	}
	return records, nil
}
