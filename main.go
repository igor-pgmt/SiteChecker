package main

import (
	"bufio"
	"encoding/csv"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"github.com/headzoo/surf"
	"golang.org/x/text/encoding/charmap"
)

// Command-line flags
var fileInput string
var fileResult string
var www int
var firstLine int
var threads int
var timeouts int
var help bool

// Link to website checking
var linkBase = "https://spywords.ru/sword.php?region=&sword="

// First row for resulting csv file
var firstRow = "№,website,yandex,google,title,website error,redirect,newWebsite,error,spywords error\n"

// checkFlags checks for required command-line flags
func checkFlags() {
	flag.StringVar(&fileInput, "fileInput", "", "Please, specify input file name  [REQUIRED]")
	flag.StringVar(&fileResult, "fileResult", "result.txt", "Please, specify resulting file name")
	flag.IntVar(&www, "www", 0, "Please, specify column with http:// web addresses [REQUIRED]")
	flag.IntVar(&firstLine, "firstLine", 0, "Please, specify first line for file to grab data")
	flag.IntVar(&threads, "threads", 1, "Please, specify amount of threads [min:1, max:100]")
	flag.IntVar(&timeouts, "timeouts", 30, "Please, specify timeout in seconds [min:1]")
	flag.BoolVar(&help, "help", false, "The help command to show this message")
	flag.Parse()

	for _, arg := range os.Args[:] {
		if arg == "-" {
			flag.Usage()
			os.Exit(1)
		}
	}

	if help || fileInput == "" || www < 0 || firstLine < 0 || threads < 1 || timeouts < 1 {
		flag.Usage()
		os.Exit(1)
	}

	if threads > 100 {
		fmt.Println("Sorry, 100 is max")
		flag.Usage()
		os.Exit(1)
	}

}

// Write(append) string to a file
func writeToFile(file *os.File, writeString string, i int) {

	// Write CSV-alike textline to the output file
	_, err := file.WriteString(writeString)
	check(err)

	// Save changes
	err = file.Sync()
	check(err)

	fmt.Printf("Line %d appended to the file %v\n", i, fileResult)
}

func main() {
	// Check for flags
	checkFlags()

	// Open CSV file
	csvFile, err := os.Open(fileInput)
	check(err)
	defer csvFile.Close()

	// Create resulting file
	file, err := os.Create(fileResult)
	check(err)
	defer file.Close()
	writeToFile(file, firstRow, 0)
	fmt.Printf("File %s created\n", fileResult)

	// Create CSV Reader
	csvReader := csv.NewReader(bufio.NewReader(csvFile))

	// Get CSV rows
	records, err := csvReader.ReadAll()
	check(err)
	fmt.Printf("Total: %v rows\n", len(records))

	// Create waitgroup for goroutines waiting
	wg := &sync.WaitGroup{}

	// Create operations limiting channel
	limitChan := make(chan struct{}, threads)

	// Iterate websites
	for i := firstLine; i < len(records); i++ {

		// Get cell content (www address)
		websiteCell := records[i][www]
		// Removing spaces
		websiteCell = strings.Replace(websiteCell, " ", "", -1)
		// Check for protocol
		checkedLink := checkProtocol(websiteCell)

		// Increment waitgroup counter
		wg.Add(1)
		// Adding "goroutine" to working slot
		limitChan <- struct{}{}
		go checkWebsite(file, limitChan, checkedLink, wg, i)
	}
	wg.Wait()
}

func checkWebsite(file *os.File, limitChan chan struct{}, websiteCell string, wg *sync.WaitGroup, i int) {

	fmt.Println("№:", i, websiteCell)
	var title, err1txt, err2txt, err3txt, redirect, newWebsite string

	// Connect to Website and get the title
	var netClient = createClient()
	res, err1 := connectToWebsite(netClient, websiteCell)
	if err1 != nil {
		err1txt = err1.Error()
	} else {
		defer res.Body.Close()
		if checkRedirect(res) {
			location := res.Header.Get("Location")
			redirect = res.Status
			newWebsite = location
			var netClient2 = createClientEnd()
			if location == websiteCell {
				res, err1 = connectToWebsite(netClient2, websiteCell)
			} else {
				var err2 error
				res, err2 = connectToWebsite(netClient2, websiteCell)
				if err2 != nil {
					err2txt = err2.Error()
				} else {
					title, _ = getTitle(res)
				}
			}
			websiteCell = checkProtocol(newWebsite)
		} else {
			title, _ = getTitle(res)
		}
	}

	// Get statistics from the spyword website
	result, err3 := getSpyWordsInfo(websiteCell, i)
	if err3 != nil {
		err3txt = err3.Error()
	}

	// String to write
	writeString := "\"" + strconv.Itoa(i) + "\"" + "," +
		"\"" + websiteCell + "\"" + "," +
		"\"" + result["yandex"] + "\"" + "," +
		"\"" + result["google"] + "\"" + "," +
		"\"" + escapeQuotes(title) + "\"" + "," +
		"\"" + escapeQuotes(err1txt) + "\"" + "," +
		"\"" + redirect + "\"" + "," +
		"\"" + newWebsite + "\"" + "," +
		"\"" + escapeQuotes(err2txt) + "\"" + "," +
		"\"" + escapeQuotes(err3txt) + "\"" +
		"\n"

	// Write results to the file
	writeToFile(file, writeString, i)
	// Release working slot
	<-limitChan
	// Decrement waitgroup counter
	wg.Done()
}

func getSpyWordsInfo(link string, i int) (map[string]string, error) {
	// Encode link in win1251
	enc := charmap.Windows1251.NewEncoder()
	link1251, _ := enc.String(linkBase + link)

	// Create new Browser
	var browser = surf.NewBrowser()
	// Open link in Browser
	err := browser.Open(link1251)
	check(err)

	// Getting needed table from the website
	table := browser.Find("table.data_table.stat")

	// Getting needed cells from the table
	td := table.Find("tr.white td")
	// Parsed content
	result := make(map[string]string)
	result["google"] = "no info"
	result["yandex"] = "no info"

	// Search for the needed information
	if td.Length() > 0 {
		// Iterate finded cells and get info
		td.Each(func(i int, s *goquery.Selection) {
			cellContent := s.Text()
			switch i {
			case 1:
				result["yandex"] = cellContent
			case 10:
				result["google"] = cellContent
			}
		})
	}

	return result, nil
}
