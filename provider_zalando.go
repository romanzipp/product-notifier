package main

import (
	"encoding/json"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"log"
	"net/http"
	"time"
)

const PROVIDER_ZALANDO = "zalando"

type Zalando struct {
	Provider
}

type ZalandoData struct {
	Threads struct {
		Products map[string]NikeProduct `json:"products"`
	} `json:"Threads"`
}

type ProductInfo struct {
	Data struct {
		Context struct {
			EntityId string `json:"entity_id"`
			Name     string `json:"name"`
			Simples  []struct {
				Size  string `json:"size"`
				Offer struct {
					Stock struct {
						Quantity string `json:"quantity"`
					} `json:"stock"`
				} `json:"offer"`
			} `json:"simples"`
			Etc map[string]interface{} `json:"-"`
		} `json:"context"`
	} `json:"data"`
}

func (provider Zalando) Check(av *Availability) {
	client := http.Client{
		Timeout: time.Second * 5,
	}

	// execute request
	req, _ := http.NewRequest(http.MethodGet, av.Provider.GetUrl(), nil)
	res, err := client.Do(req)
	if err != nil {
		log.Println("error sending request")
		log.Println(err)
		return
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		log.Fatal(err)
	}

	// Find the review items
	doc.Find("script[data-re-asset].re-1-12").Each(func(i int, s *goquery.Selection) {
		// For each item found, get the title
		html := s.Text()
		var content struct {
			GraphqlCache map[string]interface{} `json:"graphqlCache"`
		}

		// unpack data to content struct
		if err = json.Unmarshal([]byte(html), &content); err != nil {
			return
		}

		for _, value := range content.GraphqlCache {
			fmt.Println("---------------------------------------------------------")

			data, _ := json.Marshal(value)
			//fmt.Println(string(data))

			var info ProductInfo

			if err := json.Unmarshal(data, &info); err != nil {
				fmt.Println(err)
			}

			//fmt.Printf("%s\n", pi.Data.Context.EntityId)

			if info.Data.Context.EntityId == "" {
				continue
			}

			for _, s := range info.Data.Context.Simples {
				fmt.Printf("%s :: %s\n", s.Size, s.Offer.Stock.Quantity)
			}
			//fmt.Printf("\n\n%v\n", info)
		}
	})
}
