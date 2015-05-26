package main

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
)

type (
	Address struct {
		No         string `json:"no"`
		Location   string `json:"location"`
		Postcode   string `json:"postcode"`
		PostOffice string `json:"postOffice"`
		State      string `json:"state"`
	}

	Jobs struct {
		WorkList  map[string]bool
		Addresses []*Address
		*sync.Mutex
	}
)

const host = "http://www.pos.com.my"
const endpoint = host + "/postal-services/quick-access/"

func main() {
	t := runtime.NumCPU()
	runtime.GOMAXPROCS(t)

	states, err := getStates()
	if err != nil {
		fmt.Println("failed to get state options")
		os.Exit(1)
	}

	<-processState(states[0])
	// var wg sync.WaitGroup
	// statesChan := make(chan string, len(states))
	// for _, s := range states {
	// 	wg.Add(1)
	// 	statesChan <- s
	// }

	// for i := 0; i < t; i++ {
	// 	go func(statesChan chan string) {
	// 		for {
	// 			select {
	// 			case state := <-statesChan:
	// 				success := <-processState(state)
	// 				if !success {
	// 					fmt.Printf("failed while processing %s, retrying in 5 seconds...\n", state)
	// 					statesChan <- state
	// 					<-time.After(5 * time.Second)
	// 				} else {
	// 					wg.Done()
	// 				}
	// 			}
	// 		}
	// 	}(statesChan)
	// }
	// wg.Wait()
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

func processState(state string) <-chan bool {
	resp := make(chan bool, 1)
	fmt.Printf("processing... %s\n", state)

	jobs := &Jobs{
		WorkList: make(map[string]bool),
		Address:  make([]*Address, 0),
		Mutex:    &sync.Mutex{},
	}

	params := url.Values{}
	params.Set("postcodeFinderState", state)
	params.Set("postcodeFinderLocation", "")
	params.Set("page", "1")
	url := fmt.Sprintf("%s?%s", endpoint, params.Encode())
	jobs.WorkList[url] = false

	if err := processStatePage(url, jobs); err != nil {
		for u, ok := range jobs.WorkList {
			if !ok {
				if err := processState(url, jobs); err == nil {
					break
				}
			}
		}
	} else {
		resp <- true
	}

	return resp
}

func processStatePage(url string, jobs *Jobs) error {
	doc, err := download(url)
	if err != nil {
		return err
	}

	table := doc.Find("div#postcode-finder-output table")
	addresses := make([]*Address, 0)
	table.Find("tbody tr").Each(func(i int, s *goquery.Selection) {
		children := s.Children()
		address := &Address{
			No:         children.Eq(0).Text(),
			Location:   children.Eq(1).Text(),
			Postcode:   children.Eq(2).Text(),
			PostOffice: children.Eq(3).Text(),
			State:      children.Eq(4).Text(),
		}
		addresses = append(addresses, address)
	})

	jobs.Addresses = append(jobs.Addresses, addresses...)

	links := table.Find("tfoot a")
	for i := range links.Nodes {
		s := links.Eq(i)
		href, _ := s.Attr("href")
		url := host + href
		if done, found := jobs.WorkList[url]; !found {
			jobs.WorkList[url] = false
		} else {
			if done {
				// skipping this hell
				return
			}
		}
		if err := processState(url, jobs); err != nil {
			return err
		}
	}
	jobs.WorkList[url] = true
	return nil
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
