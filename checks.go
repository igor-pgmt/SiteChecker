package main

import (
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/headzoo/surf/browser"
	"golang.org/x/net/html/charset"
	"golang.org/x/text/encoding/charmap"
)

// Maintenance function for error checking
func check(err error) {
	if err != nil {
		panic(err)
	}
}

// Check for protocol http:// or https://
func checkProtocol(link string) string {
	hasProtocol := strings.HasPrefix(link, "http")
	if !hasProtocol {
		return "http://" + link
	}
	return link
}

// Check if redirect exists
func checkRedirectOld(browser *browser.Browser) string {
	// If there is "canonical" header it means redirect
	respHeaders := browser.ResponseHeaders()
	if len(respHeaders["Link"]) > 0 {
		re := regexp.MustCompile("canonical")
		match := re.MatchString(respHeaders["Link"][0])
		if match {
			re := regexp.MustCompile("\\<(.*?)\\>")
			match := re.FindStringSubmatch(respHeaders["Link"][0])
			return match[1]
		}
	}
	return ""
}

// checkRedirect checks response for redirect
func checkRedirect(res *http.Response) bool {
	redirects := []int{301, 302, 303, 307, 308}
	for _, redirect := range redirects {
		if res.StatusCode == redirect {
			return true
		}
	}
	return false
}

// connectToWebsite connects to website and returns http client
func connectToWebsite(netClient *http.Client, websiteCell string) (*http.Response, error) {
	// Create new Client
	res, err := netClient.Get(websiteCell)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// Determine encoding and convert to utf-8 if needed
func toUTF8(s *string) {
	_, name, _ := charset.DetermineEncoding([]byte(*s), "text/html")
	if name != "utf-8" {
		enc := charmap.Windows1251.NewDecoder()
		*s, _ = enc.String(*s)
	}
}

// createClient creates *http.Client with timeout and redirect checking
func createClient() *http.Client {
	return &http.Client{
		Timeout: time.Duration(timeouts) * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

// createClient creates *http.Client with timeout only
func createClientEnd() *http.Client {
	return &http.Client{
		Timeout: time.Duration(timeouts) * time.Second,
	}
}

// getTitle gets the title grom server response
func getTitle(res *http.Response) (string, error) {
	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return "", err
	}
	// Find the title
	title := doc.Find("title").Text()
	toUTF8(&title)
	return title, nil
}

// escapeQuotes escapes single qoute by adding the second quote
func escapeQuotes(s string) string {
	return strings.Replace(s, "\"", "\"\"", -1)
}
