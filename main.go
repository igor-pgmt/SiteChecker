package main

import (
	"bufio"
	"encoding/csv"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/headzoo/surf"
	"golang.org/x/text/encoding/charmap"
)

// Command-line flags
var fileInput string
var fileResult string
var www int
var firstLine int
var clean int
var help bool

// Link to website checking
var linkBase = "https://spywords.ru/sword.php?region=&sword="

// Parsed content
var google string
var yandex string
var info string

// Maintenance function for error checking
func check(err error) {
	if err != nil {
		panic(err)
	}
}

// Initial function checks for required command-line flags
func init() {
	flag.StringVar(&fileInput, "fileInput", "", "Please, specify input file name  [REQUIRED]")
	flag.StringVar(&fileResult, "fileResult", "result.txt", "Please, specify resulting file name")
	flag.IntVar(&www, "www", 0, "Please, specify column with http:// web addresses [REQUIRED]")
	flag.IntVar(&firstLine, "firstLine", 0, "Please, specify first line for file to grab data")
	flag.IntVar(&clean, "clean", 9, "Please, specify cache cleaning frequency")
	flag.BoolVar(&help, "help", false, "The help command to show this message")
	flag.Parse()

	if help || fileInput == "" || www < 0 || firstLine < 0 || clean < 1 {
		flag.Usage()
		os.Exit(1)
	}
}

func main() {
	// Open CSV file
	csvFile, err := os.Open(fileInput)
	check(err)
	defer csvFile.Close()

	// Create resulting file
	createFile(fileResult)

	// Create CSV Reader
	csvReader := csv.NewReader(bufio.NewReader(csvFile))

	// Create new Browser
	var browser = surf.NewBrowser()

	// Loop CSV results
	records, err := csvReader.ReadAll()
	check(err)
	fmt.Println(len(records))

	for i := firstLine; i < len(records); i++ {

		fmt.Println("№:", i)

		// Get cell content (www address)
		websiteCell := records[i][www]
		// Removing spaces
		websiteCell = strings.Replace(websiteCell, " ", "", -1)
		// Check for protocol
		hasProtocol := strings.HasPrefix(websiteCell, "http")
		if !hasProtocol {
			websiteCell = "http://" + websiteCell
		}

		// Website state
		infoError := browser.Open(websiteCell)
		if infoError != nil {
			info = infoError.Error()
		} else {
			info = "no error"
			fmt.Println("Title:", browser.Title())
		}

		// If we connect to spywords "clean" times, create a new browser to clean cache\cookies
		if i%clean == 0 {
			browser = surf.NewBrowser()
		}

		// Creating link to check the website
		link := linkBase + websiteCell
		fmt.Println("utf8lnk:", link)
		enc := charmap.Windows1251.NewEncoder()
		link1251, _ := enc.String(link)
		fmt.Println("win1251:", link1251) //show win-1251 encoded text

		// Open link in Browser
		err = browser.Open(link1251)
		check(err)

		// Getting needed table from the website
		table := browser.Find("table.data_table.stat")

		// Getting needed cells from the table
		td := table.Find("tr.white td")

		// Go to the next loop if there is no results
		if td.Length() < 1 {
			yandex = "no info"
			google = "no info"
			writeFile(websiteCell, yandex, google, info)
			continue
		}

		// Loop finded cells and get info
		td.Each(func(i int, s *goquery.Selection) {
			a := s.Text()

			switch i {
			case 1:
				yandex = a
			case 10:
				google = a
				writeFile(websiteCell, yandex, google, info)
				break
			}

		})
	}

}

// Create file (rewrites if exists)
func createFile(fileResult string) {
	file, err := os.Create(fileResult)
	check(err)
	defer file.Close()

	fmt.Println("File created")
}

// Write(append) string to a file
func writeFile(websiteCell, yandex, google, info string) {
	// Open file using READ & WRITE permission
	var file, err = os.OpenFile(fileResult, os.O_WRONLY|os.O_APPEND, 0644)
	check(err)
	defer file.Close()

	// Write CSV-alike textline to the output file
	writestring := "\"" + websiteCell + "\"" + "," + "\"" + yandex + "\"" + "," + "\"" + google + "\"" + ",\"" + info + "\"\n"
	_, err = file.WriteString(writestring)
	check(err)

	// Save changes
	err = file.Sync()
	check(err)

	fmt.Println("Line appended to the file", fileResult)
}
