package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const PROVIDER_NIKE = "nike"

type Nike struct {
	Provider
}

type NikeData struct {
	Threads struct {
		Products map[string]NikeProduct `json:"products"`
	} `json:"Threads"`
}

type NikeProduct struct {
	Id                string `json:"id"`
	Brand             string `json:"brand"`
	Color             string `json:"colorDescription"`
	Title             string `json:"title"`
	FullTitle         string `json:"fullTitle"`
	FirstImageUrl     string `json:"firstImageUrl"`
	LastSizeAvailable string
	Skus              []struct {
		Id                  string `json:"id"`
		NikeSize            string `json:"nikeSize"`
		SkuId               string `json:"skuId"`
		LocalizedSize       string `json:"localizedSize"`
		LocalizedSizePrefix string `json:"localizedSizePrefix"`
	} `json:"skus"`
	AvailableSkus []struct {
		Id           string `json:"id"`
		ProductId    string `json:"productId"`
		ResourceType string `json:"resourceType"`
		Available    bool   `json:"available"`
		Level        string `json:"level"`
		SkuId        string `json:"skuId"`
	} `json:"availableSkus"`
}

type TmpSize struct {
	Size      string
	Available bool
}

func (provider Nike) Check(prod *Product) {
	client := http.Client{
		Timeout: time.Second * 5,
	}

	// execute request
	req, _ := http.NewRequest(http.MethodGet, prod.Provider.GetUrl(), nil)
	res, err := client.Do(req)
	if err != nil {
		log.Println("error sending request")
		log.Println(err)
		return
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	// unpack body stream
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Println("error reading response")
		return
	}

	// parse content via regex pattern
	matches := regexp.MustCompile(`<script>window\.INITIAL_REDUX_STATE=(.*);</script>`).FindStringSubmatch(string(body))
	if len(matches) != 2 {
		log.Println("error finding website content data")
		return
	}

	content := NikeData{}

	// unpack data to content struct
	if err = json.Unmarshal([]byte(matches[1]), &content); err != nil {
		log.Println("error unmarshalling json")
		return
	}

	//log.Printf("%v", content)

	// pull the first and hopefully only product from content
	for _, vendorProduct := range content.Threads.Products {
		var tmpSizes []TmpSize

		// iterate through all listed sizes
		for _, sku := range vendorProduct.Skus {
			// iterate through all available sizes stored in another map
			for _, availableSku := range vendorProduct.AvailableSkus {
				if availableSku.SkuId == sku.SkuId {

					if sku.LocalizedSize == prod.Size.GetEuSize() {
						prod.Available = availableSku.Available

						if prod.Available != prod.PreviouslyAvailable {
							if prod.Available {
								prod.notify(true)
							} else {
								prod.notify(false)
							}

							prod.PreviouslyAvailable = prod.Available
						}
					}

					tmpSizes = append(tmpSizes, TmpSize{
						Size:      sku.LocalizedSize,
						Available: availableSku.Available,
					})
				}
			}
		}

		var avs []string
		for _, size := range tmpSizes {
			if size.Available {
				avs = append(avs, size.Size)
			}
		}

		prod.Log(fmt.Sprintf("found %s\n", strings.Join(avs, ", ")))

		// the products map should only contain one index, so fuck everything else
		break
	}

}
