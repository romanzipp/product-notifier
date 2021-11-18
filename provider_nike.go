package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"time"
)

const ProviderNike = "nike"

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

type NikeSize struct {
	Size      string
	Available bool
}

func (provider Nike) GetAvailableSizes() ([]string, error) {
	client := http.Client{
		Timeout: time.Second * 5,
	}

	// execute request
	req, _ := http.NewRequest(http.MethodGet, provider.GetUrl(), nil)
	res, err := client.Do(req)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("error sending request: %s", err.Error()))
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	// unpack body stream
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, errors.New("error reading response")
	}

	// parse content via regex pattern
	matches := regexp.MustCompile(`<script>window\.INITIAL_REDUX_STATE=(.*);</script>`).FindStringSubmatch(string(body))
	if len(matches) != 2 {
		return nil, errors.New("error finding website content data")
	}

	content := NikeData{}

	// unpack data to content struct
	if err = json.Unmarshal([]byte(matches[1]), &content); err != nil {
		return nil, errors.New("error unmarshalling json")
	}

	var availableSizes []string

	// pull the first and hopefully only product from content
	for _, vendorProduct := range content.Threads.Products {
		// iterate through all listed sizes
		for _, sku := range vendorProduct.Skus {
			// iterate through all available sizes stored in another map
			for _, availableSku := range vendorProduct.AvailableSkus {
				// skip if available sku does not match to iterating sku
				if availableSku.SkuId != sku.SkuId {
					continue
				}

				if availableSku.Available {
					availableSizes = append(availableSizes, sku.LocalizedSize)
				}
			}
		}

		// the products map should only contain one index, so fuck everything else
		break
	}

	return availableSizes, nil
}
