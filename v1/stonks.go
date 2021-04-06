package stonks

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	finnhub "github.com/Finnhub-Stock-API/finnhub-go"
	"github.com/antihax/optional"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
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
	Fh            *finnhub.DefaultApiService
	finnhubApiKey string
	Fhauth        context.Context
	Records       [][]string
	DataPath      string
}

// json for short interest api

type ShortInterestResponse struct {
	Data []struct {
		Date          string `json:"date"`
		ShortInterest int    `json:"shortInterest"`
	} `json:"data"`
	Symbol string `json:"symbol"`
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
		Fh:            finnhubClient,
		Fhauth:        finnhubAuth,
		finnhubApiKey: finnhubApiKey,
		Records:       records,
		DataPath:      stonksDataPath,
	}
	return &client
}

func (s *StonksClient) ReloadDescriptions() error {
	records, err := GetStonksDataFromCSV(s.DataPath)
	if err != nil {
		log.Printf("Unable to reload descriptions %s\n", err)
		return err
	} else {
		s.Records = records
		return nil
	}

}

func (s *StonksClient) PullNewDescriptions() error {
	cmd := "/get_stonks_db.sh"
	if err := exec.Command(cmd).Run(); err != nil {
		log.Printf("Error pulling new descriptions\n")
		log.Println(os.Stderr, err)
		return err
	}
	return nil
}

func (s *StonksClient) ZQuote(symbol string) (float32, error) {
	log.Printf("Looking up stock quote: %s\n", symbol)
	quote, _, err := s.Fh.Quote(s.Fhauth, symbol)
	if err != nil {
		return 0.0, err
	}
	if quote.Pc == 0 && quote.O == 0 {
		msg := fmt.Sprintf("No data found for symbol %s\n", symbol)
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

func (s *StonksClient) CompanyProfile2(sym string) (profile finnhub.CompanyProfile2, err error) {
	// Company profile2
	symbol := strings.ToUpper(sym)

	profile2, _, err := s.Fh.CompanyProfile2(s.Fhauth, &finnhub.CompanyProfile2Opts{Symbol: optional.NewString(symbol)})
	if err != nil {
		return finnhub.CompanyProfile2{}, err
	}
	return profile2, nil
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

func (s *StonksClient) GetShortInterestBeta(symbol string) (*ShortInterestResponse, error) {

	//var result map[string]interface{}
	parsed := ShortInterestResponse{}
	dt := time.Now()
	previousDate := time.Date(dt.Year()-1, dt.Month(), dt.Day(), 0, 0, 0, 0, time.UTC)
	prevDateStr := previousDate.Format("2006-01-02")
	todayDateStr := dt.Format("2006-01-02")

	client := http.Client{}
	urlRoot := "https://finnhub.io/api/v1/"
	path := fmt.Sprintf("stock/short-interest?symbol=%s&from=%s&to=%s&token=%s", symbol, prevDateStr, todayDateStr, s.finnhubApiKey)

	fmt.Println(path)
	request, err := http.NewRequest("GET", urlRoot+path, nil)
	if err != nil {
		return &parsed, err
	}

	resp, err := client.Do(request)
	if err != nil {
		return &parsed, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	err = json.Unmarshal(body, &parsed)
	if err != nil {
		log.Fatal(err)
	}

	return &parsed, nil

}
