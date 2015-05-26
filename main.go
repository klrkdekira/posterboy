package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

const endpoint = "http://www.pos.com.my/postal-services/quick-access"

func main() {
	states, err := getStates()
	if err != nil {
		fmt.Println("failed to get state options")
		os.Exit(1)
	}

	statesChan := make(chan string)
	for _, s := range states {
		statesChan <- s
	}

	for _ = range states {
		fmt.Println(<-statesChan)
	}

	// fmt.Print(states)
}

func getStates() ([]string, error) {
	states := make([]string, 0)

	url := fmt.Sprintf("%s?postcode-finder", endpoint)
	doc, err := download(url)
	if err != nil {
		return states, err
	}

	uniqueCell := map[string]struct{}{}

	slct := doc.Find("select#postcode-finder-state-select").First()
	slct.Find("option:not([selected])").Each(func(i int, s *goquery.Selection) {
		uniqueCell[strings.Trim(s.Text(), " ")] = struct{}{}
	})

	for k, _ := range uniqueCell {
		states = append(states, k)
	}

	return states, nil
}

func download(url string) (*goquery.Document, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "posterchild")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return goquery.NewDocumentFromResponse(resp)
}
