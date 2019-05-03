package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"sync"

	"github.com/PuerkitoBio/goquery"
)

/*
Prework : mini web crawler
1. craw and parse URL (as parameter) -> url / title / author / date (loop for other URLs and again if linked & existed)
2. output to CSV file
*/
type Result struct {
	URL    string
	Title  string
	Author string
	Date   string
}

var visited = make(map[string]bool)

type CsvWriter struct {
	mutex     *sync.Mutex
	csvWriter *csv.Writer
}

func main() {

	flag.Parse()
	args := flag.Args()

	if len(args) < 1 {
		fmt.Println("please specify start page")
		os.Exit(1)
	}

	inputQueue := make(chan string)
	outputQueue := make(chan Result)

	go func() {
		inputQueue <- args[0]
	}()

	for uri := range inputQueue {
		enqueue(uri, inputQueue, outputQueue)
	}

	writer, err := newCsvWriter("/tmp/output.csv")
	if err != nil {
		fmt.Println("can not create output file")
		log.Panic(err)
	}
	defer writer.flush()

	go func() {
		result := <-outputQueue
		writer.write([]string{result.URL, result.Title, result.Author, result.Date})
	}()

}

func newCsvWriter(fileName string) (*CsvWriter, error) {
	csvFile, err := os.Create(fileName)
	if err != nil {
		return nil, err
	}
	writer := csv.NewWriter(csvFile)

	return &CsvWriter{csvWriter: writer, mutex: &sync.Mutex{}}, nil
}

func (w *CsvWriter) write(row []string) {
	w.mutex.Lock()
	w.csvWriter.Write(row)
	w.mutex.Unlock()
}

func (w *CsvWriter) flush() {
	w.mutex.Lock()
	w.csvWriter.Flush()
	w.mutex.Unlock()
}

func enqueue(uri string, inputQueue chan string, outputQueue chan Result) {

	fmt.Println("fetching ... ", uri)

	visited[uri] = true

	res, err := http.Get(uri)
	if err != nil {
		return
	}
	defer res.Body.Close()

	doc, err := goquery.NewDocumentFromResponse(res)
	if err != nil {
		return
	}

	result := parsePage(doc)
	go func() {
		outputQueue <- result
	}()

	links := extractLinks(doc)
	for _, link := range links {
		absolute := fixURL(link, uri)
		if absolute != "" {
			if !visited[absolute] {
				go func() {
					inputQueue <- absolute
				}()
			}
		}
	}
}

func parsePage(doc *goquery.Document) Result {
	result := Result{"empty", "empty", "empty", "empty"}

	result.URL = doc.Url.String()
	doc.Find(".the-article-title").Each(func(i int, s *goquery.Selection) {
		result.Title = s.Text()
	})
	doc.Find(".author").Each(func(i int, s *goquery.Selection) {
		result.Author = s.Text()
	})
	doc.Find(".the-article-publish").Each(func(i int, s *goquery.Selection) {
		result.Date = s.Text()
	})
	fmt.Printf("result parse is %+v \n", result)

	return result
}

func extractLinks(doc *goquery.Document) []string {
	foundURLs := []string{}

	if doc != nil {
		doc.Find("a").Each(func(i int, s *goquery.Selection) {
			res, _ := s.Attr("href")
			foundURLs = append(foundURLs, res)
		})
	}

	return foundURLs
}

func fixURL(href, base string) string {
	uri, err := url.Parse(href)
	if err != nil {
		return ""
	}

	baseURL, err := url.Parse(base)
	if err != nil {
		return ""
	}

	uri = baseURL.ResolveReference(uri)
	return uri.String()
}
