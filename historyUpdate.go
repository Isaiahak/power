package main

import (
	"fmt"
	"html"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
)

var dir = "./HistoricalPrices/"

type DailyInformation struct {
	Date   string
	Close  float64
	Volume int
	Open   float64
	High   float64
	Low    float64
}

func update(test bool) error {
	var csv []byte
	var err error
	if test {
		csv, err = os.ReadFile("tester.txt")
		if err != nil {
			return fmt.Errorf("failed to open companies: %s", err)
		}
	} else {
		csv, err = os.ReadFile("companies.txt")
		if err != nil {
			return fmt.Errorf("failed to open companies: %s", err)
		}
	}

	_, after, _ := strings.Cut(string(csv), "--Stocks--")
	stocks, _, _ := strings.Cut(after, "--ETF--")
	stocks = strings.TrimSpace(stocks)
	for stock := range strings.SplitSeq(stocks, "\n") {
		stockInfo := strings.Split(stock, ":")
		err := updateHistory(stockInfo)
		if err != nil {
			fmt.Println(err)
		}
	}
	return nil
}

func updateHistory(stockInfo []string) error {
	if len(stockInfo) > 1 {
		//stockName := stockInfo[0]
		stockSymbol := stockInfo[1]
		info, err := getStockInfo(stockSymbol)
		if err != nil {
			return fmt.Errorf("failed to retrieve stock information: %s", err)
		}
		fmt.Println(info)

		/*
			err = saveToCSV(info, stockName)
			if err != nil {
				return fmt.Errorf("failed to retieve stock information: %s", err)
			}
		*/

		return nil
	} else {
		return fmt.Errorf("stockInfo was missing information")
	}
}

func getStockInfo(stockSymbol string) (DailyInformation, error) {
	var info DailyInformation
	resp, err := http.Get("http://search.brave.com/search?q=" + stockSymbol)
	if err != nil {
		return info, fmt.Errorf("failed to search for symbol: %s", err)
	}

	if resp.StatusCode != 200 {
		return info, fmt.Errorf("request failed due to: %s", resp.Status)
	} else {
		dataRaw, err := io.ReadAll(resp.Body)
		if err != nil {
			return info, fmt.Errorf("failed to read the response body: %s", err)
		}
		data := html.EscapeString(string(dataRaw))
		startIndicator := "Open"
		_, start, _ := strings.Cut(data, startIndicator)
		endIndicator := "low"
		stockData, _, _ := strings.Cut(start, endIndicator)
		fmt.Println(stockData)
		html.Parse(stockData)

		// everything but the date is contained in this block of data
	}

	return info, nil
}

func saveToCSV(info DailyInformation, stockName string) error {

	filePath := dir + stockName + ".csv"
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open csv for %s,:\n due to: %s", stockName, err)
	}

	Volume := strconv.Itoa(info.Volume)
	Close := strconv.FormatFloat(info.Close, 'b', 3, 64)
	Open := strconv.FormatFloat(info.Open, 'b', 3, 64)
	High := strconv.FormatFloat(info.High, 'b', 3, 64)
	Low := strconv.FormatFloat(info.Low, 'b', 3, 64)

	data := []string{info.Date, Volume, Close, Open, High, Low}

	dataCSVFormat := strings.Join(data, ",")

	if _, err := f.Write([]byte(dataCSVFormat)); err != nil {
		f.Close()
		return fmt.Errorf("failed to save info to file: %s", err)
	}

	if err := f.Close(); err != nil {
		return fmt.Errorf("failed to close file: %s", err)
	}
	return nil
}
