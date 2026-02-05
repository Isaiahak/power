package main

import (
	"fmt"
	_ "golang.org/x/net/html"
	"io"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"
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
		stockName := stockInfo[0]
		// bwt, brook
		flipCSV(stockName)
		fmt.Println("finished flipped for: ", stockName)

		/*
			stockSymbol := stockInfo[1]
			info, err := getStockInfo(stockSymbol)
			if err != nil {
				return fmt.Errorf("failed to retrieve stock information: %s", err)
			}
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
		startIndicator := `<div class="flex-hcenter-apart svelte-5dzuzo">`
		_, start, _ := strings.Cut(string(dataRaw), startIndicator)

		endIndicator := "low"
		stockInfo, _, _ := strings.Cut(start, endIndicator)

		info, err = parseData(stockInfo)
		if err != nil {
			return info, fmt.Errorf("failed to parse the stock info: %s", err)
		}

		loc, _ := time.LoadLocation("America/New_York")
		t := time.Now().In(loc)
		date := t.String()
		dateInfo := strings.Split(strings.Split(date, " ")[0], "-")
		date = dateInfo[1] + "/" + dateInfo[2] + "/" + dateInfo[0]
		info.Date = date

	}

	return info, nil
}

func saveToCSV(info DailyInformation, stockName string) error {

	filePath := dir + stockName + ".csv"
	f, err := os.OpenFile(filePath, os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("failed to open csv for %s,:\n due to: %s", stockName, err)
	}

	Volume := strconv.Itoa(info.Volume)
	Close := strconv.FormatFloat(info.Close, 'f', 2, 64)
	Close = "$" + Close
	Open := strconv.FormatFloat(info.Open, 'f', 2, 64)
	Open = "$" + Open
	High := strconv.FormatFloat(info.High, 'f', 2, 64)
	High = "$" + High
	Low := strconv.FormatFloat(info.Low, 'f', 2, 64)
	Low = "$" + Low

	data := []string{info.Date, Volume, Close, Open, High, Low, "\n"}

	dataCSVFormat := strings.Join(data, ",")
	dataCSVFormat = dataCSVFormat[:len(dataCSVFormat)-2]

	if _, err := f.Write([]byte(dataCSVFormat)); err != nil {
		f.Close()
		return fmt.Errorf("failed to save info to file: %s", err)
	}

	if err := f.Close(); err != nil {
		return fmt.Errorf("failed to close file: %s", err)
	}
	return nil
}

func parseData(data string) (DailyInformation, error) {
	var info DailyInformation
	var inSpan bool
	var inKey bool
	var inValue bool
	var readingData bool
	var key strings.Builder
	var value strings.Builder
	index := 0
	for index < len(data)-1 {
		if readingData {
			if inKey && inValue == false {
				if data[index] == '<' {
					index += 6
					inValue = true
					inKey = false
					readingData = false
				} else {
					key.WriteByte(data[index])
					index++
				}
			} else if inValue {
				if data[index] == '<' {
					index += 6
					info, err := checkKeyType(&info, key, value)
					if err != nil {
						return info, err
					}
					inValue = false
					readingData = false
					key.Reset()
					value.Reset()
				} else {
					value.WriteByte(data[index])
					index++
				}
			}
		} else if inSpan {
			if data[index] == '>' {
				readingData = true
				inSpan = false
			}
			index++
		} else {
			if strings.HasPrefix(data[index:], "<span") {
				index += 12
				inSpan = true
				if strings.HasPrefix(data[index:], `"k-label`) {
					inKey = true
				} else if strings.HasPrefix(data[index:], `"k-value`) {
					inValue = true
				} else {
					inSpan = false
				}
			} else {
				index++
			}
		}
	}
	return info, nil
}

func checkKeyType(info *DailyInformation, key, value strings.Builder) (DailyInformation, error) {
	keyType := key.String()
	valueString := value.String()
	fmt.Println("key: ", keyType)
	fmt.Println("value: ", valueString)
	switch keyType {
	case "Open":
		Open, err := strconv.ParseFloat(valueString, 64)
		if err != nil {
			return *info, fmt.Errorf("failed to parse Open value to float: %s", err)
		}
		info.Open = Open
	case "High":
		High, err := strconv.ParseFloat(valueString, 64)
		if err != nil {
			return *info, fmt.Errorf("failed to parse High value to float: %s", err)
		}
		info.High = High
	case "Low":
		Low, err := strconv.ParseFloat(valueString, 64)
		if err != nil {
			return *info, fmt.Errorf("failed to parse Low value to float: %s", err)
		}
		info.Low = Low
	case "Volume":
		volume := strings.Split(valueString, ".")
		number := volume[1][len(volume[1])-1]
		var zero string
		switch number {
		case 'K':
			zero = "000"
		case 'M':
			zero = "000000"
		case 'B':
			zero = "000000000"
		case 'T':
			zero = "000000000000"
		}
		valueString = volume[0] + volume[1][:len(volume[1])-1] + zero
		Volume, err := strconv.Atoi(valueString)
		fmt.Println(Volume)
		if err != nil {
			return *info, fmt.Errorf("failed to parse Volume value to float: %s", err)
		}
		info.Volume = Volume
	case "Prev close":
		Close, err := strconv.ParseFloat(valueString, 64)
		if err != nil {
			return *info, fmt.Errorf("failed to parse Close value to float: %s", err)
		}
		info.Close = Close
	default:
	}
	return *info, nil
}

func flipCSV(stockName string) {

	filePath := dir + stockName + ".csv"
	content, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Printf("failed to readf csv for %s,:\n due to: %s", stockName, err)
	}

	contentString := strings.TrimSpace(string(content))
	contentLines := strings.Split(contentString, "\n")

	header := []string{contentLines[0]}

	contentLines = contentLines[1:]

	slices.Reverse(contentLines)

	contentLines = append(header, contentLines...)
	content = []byte(strings.Join(contentLines, "\n"))
	os.WriteFile(filePath, content, 0644)

}
