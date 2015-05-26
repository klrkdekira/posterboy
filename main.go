package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/PuerkitoBio/goquery"
)

const endpoint = "http://www.pos.com.my/postal-services/quick-access"

func main() {
	_, err := getStates()
	if err != nil {
		fmt.Println("failed to get state options")
		os.Exit(1)
	}
	// fmt.Print(states)
}

func getStates() ([]string, error) {
	states := make([]string, 0)

	url := fmt.Sprintf("%s?postcode-finder", endpoint)
	body, err := download(url)
	if err != nil {
		return states, err
	}

	doc, err := goquery.NewDocument(string(body))
	if err != nil {
		log.Print(err)
		return states, err
	}

	// options := doc.Find("select#postcode-finder-state-select option")
	options := doc.Find("option")
	for n := range options.Nodes {
		fmt.Println(n)
		fmt.Println("!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")
	}
	// fmt.Println(options.Nodes)
	// for n := range options.Nodes {

	// }
	// .Each(func(i int, s *goquery.Selection) {
	// 	fmt.Println(s)
	// })

	return states, nil
}

func download(url string) ([]byte, error) {
	var body []byte
	var err error

	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return body, err
	}
	req.Header.Set("User-Agent", "posterchild")

	resp, err := client.Do(req)
	if err != nil {
		return body, err
	}
	defer resp.Body.Close()

	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return body, err
	}
	return body, err
}
