package main

import (
	"bufio"
	"encoding/csv"
	"flag"
	"fmt"
	"net/http"
	"os"
	"regexp"
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
var clean int
var help bool

// Link to website checking
var linkBase = "https://spywords.ru/sword.php?region=&sword="

// Maintenance function for error checking
func check(err error) {
	if err != nil {
		panic(err)
	}
}

// checkFlags checks for required command-line flags
func checkFlags() {
	flag.StringVar(&fileInput, "fileInput", "", "Please, specify input file name  [REQUIRED]")
	flag.StringVar(&fileResult, "fileResult", "result.txt", "Please, specify resulting file name")
	flag.IntVar(&www, "www", 0, "Please, specify column with http:// web addresses [REQUIRED]")
	flag.IntVar(&firstLine, "firstLine", 0, "Please, specify first line for file to grab data")
	flag.IntVar(&threads, "threads", 1, "Please, specify amount of threads [min:1]")
	flag.IntVar(&clean, "clean", 9, "Please, specify cache cleaning frequency")
	flag.BoolVar(&help, "help", false, "The help command to show this message")
	flag.Parse()

	for _, arg := range os.Args[:] {
		if arg == "-" {
			flag.Usage()
			os.Exit(1)
		}
	}

	if help || fileInput == "" || www < 0 || firstLine < 0 || clean < 1 || threads < 1 {
		flag.Usage()
		os.Exit(1)
	}

}

// Check for protocol
func checkProtocol(link *string) {
	hasProtocol := strings.HasPrefix(*link, "http")
	if !hasProtocol {
		*link = "http://" + *link
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
		checkProtocol(&websiteCell)

		// Increment waitgroup counter
		wg.Add(1)
		// Adding "goroutine" to working slot
		limitChan <- struct{}{}
		go checkWebsite(file, limitChan, websiteCell, wg, i)

	}
	wg.Wait()

}

func checkWebsite(file *os.File, limitChan chan struct{}, websiteCell string, wg *sync.WaitGroup, i int) {

	fmt.Println("№:", i)
	// The main part of the link
	link := linkBase
	// New address link
	var redirectLink string
	// Response headers
	var respHeaders http.Header
	// Website state info
	var info string

	// Create new Browser
	var browser = surf.NewBrowser()

	// Website state
	infoError := browser.Open(websiteCell)
	if infoError != nil {
		info = infoError.Error()
	} else {
		info = "no error"
		fmt.Println("Title:", browser.Title())
		fmt.Println("websiteCell =", websiteCell)

		// If there is "canonical" header it means redirect
		respHeaders = browser.ResponseHeaders()
		if len(respHeaders["Link"]) > 0 {
			re := regexp.MustCompile("canonical")
			match := re.MatchString(respHeaders["Link"][0])
			if match {
				re := regexp.MustCompile("\\<(.*?)\\>")
				match := re.FindStringSubmatch(respHeaders["Link"][0])
				redirectLink = match[1]
				// Creating link to check the website
				link += redirectLink
			}
		} else {
			// Creating link to check the website
			link += websiteCell
		}
	}

	// Encode link in win1251
	fmt.Println("utf8lnk:", link)
	enc := charmap.Windows1251.NewEncoder()
	link1251, _ := enc.String(link)
	fmt.Println("win1251:", link1251) // Shows win-1251 encoded text

	// Open link in Browser
	err := browser.Open(link1251)
	check(err)

	// Getting needed table from the website
	table := browser.Find("table.data_table.stat")

	// Getting needed cells from the table
	td := table.Find("tr.white td")

	// Parsed content
	var google = "no info"
	var yandex = "no info"

	// Search for the needed information
	if td.Length() > 0 {
		// Iterate finded cells and get info
		td.Each(func(i int, s *goquery.Selection) {
			cellContent := s.Text()
			switch i {
			case 1:
				yandex = cellContent
			case 10:
				google = cellContent
				// break
			}
		})
	}

	// Row assembly
	writeString := "\"" + strconv.Itoa(i) + "\"" + "," + "\"" + websiteCell + "\"" + "," + "\"" + yandex + "\"" + "," + "\"" + google + "\"" + ",\"" + info + "\""
	if redirectLink != "" {
		writeString += ",\"" + redirectLink + "\""
		writeString += ",\"" + fmt.Sprint(respHeaders) + "\""
	}
	writeString += "\n"

	// Write results to the file
	writeToFile(file, writeString, i)
	// Release working slot
	<-limitChan
	// Decrement waitgroup counter
	wg.Done()

}
