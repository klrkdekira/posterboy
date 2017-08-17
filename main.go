package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

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
		Addresses []*Address
		CheckList map[string]bool
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

	// yeah I said so
	os.MkdirAll("downloads", 0755)

	var wg sync.WaitGroup
	statesChan := make(chan string, len(states))
	for _, s := range states {
		wg.Add(1)
		statesChan <- s
	}

	for i := 0; i < t; i++ {
		go func(statesChan chan string) {
			for {
				select {
				case state := <-statesChan:
					success := <-processState(state)
					if !success {
						fmt.Printf("failed while processing %s, retrying in 5 seconds...\n", state)
						statesChan <- state
						<-time.After(5 * time.Second)
					} else {
						wg.Done()
					}
				}
			}
		}(statesChan)
	}
	wg.Wait()
}

func getStates() ([]string, error) {
	states := make([]string, 0)

	target := fmt.Sprintf("%s?postcode-finder", endpoint)
	doc, err := download(target)
	if err != nil {
		return states, err
	}

	uniqueCell := map[string]struct{}{}

	slct := doc.Find("select#find-outlet-state-select").First()
	slct.Find("option:not([selected])").Each(func(i int, s *goquery.Selection) {
		uniqueCell[strings.Trim(s.Text(), " ")] = struct{}{}
	})

	for k, _ := range uniqueCell {
		states = append(states, k)
	}

	return states, nil
}

func processState(state string) <-chan bool {
	fmt.Printf("processing... %s\n", state)

	jobs := &Jobs{
		Addresses: make([]*Address, 0),
		CheckList: make(map[string]bool),
		Mutex:     &sync.Mutex{},
	}

	params := url.Values{}
	params.Set("postcodeFinderState", state)
	params.Set("postcodeFinderLocation", "")
	params.Set("page", "1")
	target := fmt.Sprintf("%s?%s", endpoint, params.Encode())

	jobs.CheckList[target] = false
	for {
		if err := jobs.execute(target); err == nil {
			break
		}
	}

	resp := make(chan bool, 1)

	b, err := json.Marshal(jobs.Addresses)
	if err != nil {
		fmt.Println(err)
		resp <- false
		return resp
	}

	if err := ioutil.WriteFile(fmt.Sprintf("downloads/%s.json", state), b, 0755); err != nil {
		fmt.Println(err)
		resp <- false
		return resp
	}

	resp <- true
	return resp
}

func (j *Jobs) execute(target string) error {
	// you wish
	// <-time.After(1 * time.Second)
	fmt.Printf("doing %s...\n", target)
	doc, err := download(target)
	if err != nil {
		return err
	}

	table := doc.Find("div#postcode-finder-output table")
	addresses := make([]*Address, 0)
	table.Find("tbody tr").Each(func(i int, s *goquery.Selection) {
		children := s.Children()
		// TODO need to trim these hairy dudes
		address := &Address{
			No:         children.Eq(0).Text(),
			Location:   children.Eq(1).Text(),
			Postcode:   children.Eq(2).Text(),
			PostOffice: children.Eq(3).Text(),
			State:      children.Eq(4).Text(),
		}
		addresses = append(addresses, address)
	})

	j.Mutex.Lock()
	j.Addresses = append(j.Addresses, addresses...)
	j.CheckList[target] = true
	j.Mutex.Unlock()

	links := table.Find("tfoot a")
	for i := range links.Nodes {
		s := links.Eq(i)
		href, _ := s.Attr("href")
		newTarget := host + href

		if _, found := j.CheckList[newTarget]; !found {
			j.Mutex.Lock()
			j.CheckList[newTarget] = false
			j.Mutex.Unlock()
			if err := j.execute(newTarget); err != nil {
				return err
			}
		}
	}

	return nil
}

func download(target string) (*goquery.Document, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", target, nil)
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
