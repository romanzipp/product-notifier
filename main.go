package main

import (
	"encoding/json"
	"fmt"
	gosxnotifier "github.com/deckarep/gosx-notifier"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const url = "https://www.nike.com/de/t/air-force-1-luxe-herrenschuh-86CTL1"

type Content struct {
	Threads struct {
		Products struct {
			AirForceOne struct {
				Skus []struct {
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
			} `json:"DD9605-100"`
		} `json:"products"`
	} `json:"Threads"`
}

type Size struct {
	Id        string
	NikeSize  string
	EuSize    string
	Available bool
}

func notify(size Size){
	fmt.Println(strings.Repeat("#", 120))
	fmt.Println(strings.Repeat("#", 120))
	fmt.Printf("\n  %s VERFÜGBAR \n\n", size.EuSize)
	fmt.Println(strings.Repeat("#", 120))
	fmt.Println(strings.Repeat("#", 120))

	note := gosxnotifier.NewNotification(fmt.Sprintf("Größe %s jetzt verfügbar", size.EuSize))
	note.Title = "Nike Air Force VERFÜGBAR 👟"
	note.Sound = gosxnotifier.Sosumi
	note.Group = "com.nike.go"
	note.Sender = "com.apple.Safari"
	note.Link = url

	if err := note.Push(); err != nil {
		log.Println("Uh oh!")
	}
}

func main(){
	for {
		check()
		time.Sleep(10 * time.Second)
	}
}

func check() {
	search := map[string]bool{
		"44":   true,
		"44.5": true,
		//"39": true,
	}

	client := http.Client{
		Timeout: time.Second * 5,
	}

	req, _ := http.NewRequest(http.MethodGet, url, nil)

	res, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatal(err)
	}

	reg := regexp.MustCompile(`<script>window\.INITIAL_REDUX_STATE=(.*);</script>`)

	x := reg.FindStringSubmatch(string(body))
	if len(x) != 2 {
		log.Fatalln("oh no")
	}
	jc := x[1]

	data := Content{}

	if err = json.Unmarshal([]byte(jc), &data); err != nil {
		log.Fatal(err)
	}

	var sizes []Size

	for _, sku := range data.Threads.Products.AirForceOne.Skus {

		tsize := Size{
			Id:       sku.SkuId,
			NikeSize: sku.NikeSize,
			EuSize:   sku.LocalizedSize,
		}

		for _, asku := range data.Threads.Products.AirForceOne.AvailableSkus {
			if asku.SkuId == tsize.Id {
				tsize.Available = asku.Available
				break
			}
		}

		sizes = append(sizes, tsize)
	}

	var found bool

	for _, size := range sizes {
		if search[size.EuSize] && size.Available {
			notify(size)
			found = true
		}
	}

	if !found {
		var avs []string
		for _, size := range sizes {
			if size.Available {
				avs = append(avs, size.EuSize)
			}
		}

		fmt.Printf("out of stock... (%s)\n", strings.Join(avs, ", "))
	}
}
