package main

import (
	"encoding/json"
	"fmt"
	gosxnotifier "github.com/deckarep/gosx-notifier"
	"github.com/gregdel/pushover"
	"github.com/joho/godotenv"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

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

func notify(size Size) {
	msg := struct {
		Title string
		Body  string
		Url   string
	}{"Nike Air Force VERFÜGBAR 👟", fmt.Sprintf("Größe %s jetzt verfügbar", size.EuSize), os.Getenv("NIKE_URL")}

	fmt.Println(strings.Repeat("#", 120))
	fmt.Println(strings.Repeat("#", 120))
	fmt.Printf("\n  %s\n", msg.Title)
	fmt.Printf("  %s\n\n", msg.Body)
	fmt.Println(strings.Repeat("#", 120))
	fmt.Println(strings.Repeat("#", 120))

	go func() {
		note := gosxnotifier.NewNotification(msg.Body)
		note.Title = msg.Title
		note.Sound = gosxnotifier.Sosumi
		note.Group = "com.nike.go"
		note.Sender = "com.apple.Safari"
		note.Link = os.Getenv("NIKE_URL")

		if err := note.Push(); err != nil {
			log.Println("error sending macos notification")
			log.Println(err)
		}
	}()

	go func() {
		app := pushover.New(os.Getenv("PUSHOVER_APP_TOKEN"))
		recipient := pushover.NewRecipient(os.Getenv("PUSHOVER_USER_TOKEN"))

		message := pushover.NewMessage(msg.Body)
		message.Title = msg.Title
		message.URL = msg.Url

		if _, err := app.SendMessage(message, recipient); err != nil {
			log.Println("error sending pushover notification")
			log.Println(err)
		}
	}()
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	sizes := strings.Split(os.Getenv("SIZES"), ",")

	log.Printf("Searching for sizes %s\n", strings.Join(sizes, ", "))

	for {
		check(sizes)
		time.Sleep(10 * time.Second)
	}
}

func check(csizes []string) {
	search := make(map[string]bool)

	for _, csize := range csizes {
		search[csize] = true
	}

	client := http.Client{
		Timeout: time.Second * 5,
	}

	req, _ := http.NewRequest(http.MethodGet, os.Getenv("NIKE_URL"), nil)

	res, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Println("error reading response")
		return
	}

	reg := regexp.MustCompile(`<script>window\.INITIAL_REDUX_STATE=(.*);</script>`)

	x := reg.FindStringSubmatch(string(body))
	if len(x) != 2 {
		log.Println("error finding website content data")
		return
	}
	jc := x[1]

	data := Content{}

	if err = json.Unmarshal([]byte(jc), &data); err != nil {
		log.Println("error unmarshalling json")
		return
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
