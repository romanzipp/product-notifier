package main

import (
	"encoding/json"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"log"
	"net/http"
	"strings"
	"time"
)

const PROVIDER_ZALANDO = "zalando"

type Zalando struct {
	Provider
}

type ZalandoData struct {
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
			data, _ := json.Marshal(value)
			var info ZalandoData

			if err := json.Unmarshal(data, &info); err != nil {
				fmt.Println(err)
			}

			if info.Data.Context.EntityId == "" {
				continue
			}

			for _, zalSize := range info.Data.Context.Simples {
				for _, size := range av.Sizes {
					if size.EuSize == zalSize.Size {
						size.Available = zalSize.Offer.Stock.Quantity != "OUT_OF_STOCK"

						if size.Available != size.PreviouslyAvailable {
							if size.Available {
								av.notify(size, true)
							} else {
								av.notify(size, false)
							}

							size.PreviouslyAvailable = size.Available
						}
					}
				}
			}

			var logStr []string
			for _, zalSize := range info.Data.Context.Simples {
				if zalSize.Offer.Stock.Quantity != "OUT_OF_STOCK" {
					logStr = append(logStr, zalSize.Size)
				}
			}

			av.Log(fmt.Sprintf("found %s\n", strings.Join(logStr, ", ")))
		}
	})
}
