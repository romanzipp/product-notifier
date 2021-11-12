package main

import (
	"encoding/json"
	"fmt"
	gosxnotifier "github.com/deckarep/gosx-notifier"
	"github.com/gregdel/pushover"
	"github.com/joho/godotenv"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Product struct {
	Url           string
	Sizes         []*Size
	VendorProduct VendorProduct
}

type Size struct {
	Id                  string
	NikeSize            string
	EuSize              string
	Available           bool
	PreviouslyAvailable bool
}

type VendorData struct {
	Threads struct {
		Products map[string]VendorProduct `json:"products"`
	} `json:"Threads"`
}

type VendorProduct struct {
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

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	var products []*Product

	cfgSizes := strings.Split(os.Getenv("SIZES"), ",")
	cfgUrls := strings.Split(os.Getenv("NIKE_URLS"), ",")

	for _, url := range cfgUrls {
		product := Product{
			Url: url,
		}

		for _, size := range cfgSizes {
			product.Sizes = append(product.Sizes, &Size{
				EuSize: size,
			})
		}

		products = append(products, &product)
	}

	interval, _ := strconv.ParseInt(os.Getenv("INTERVAL"), 10, 0)
	log.Printf("looping in %d seconds\n", interval)

	for {
		for _, prod := range products {
			check(prod)
		}

		time.Sleep(time.Duration(interval) * time.Second)
	}
}

func check(prod *Product) {
	client := http.Client{
		Timeout: time.Second * 5,
	}

	// execute request
	req, _ := http.NewRequest(http.MethodGet, prod.Url, nil)
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

	content := VendorData{}

	// unpack data to content struct
	if err = json.Unmarshal([]byte(matches[1]), &content); err != nil {
		log.Println("error unmarshalling json")
		return
	}

	//log.Printf("%v", content)

	// pull the first and hopefully only product from content
	for _, vendorProduct := range content.Threads.Products {
		prod.VendorProduct = vendorProduct

		// the products map should only contain one index, so fuck everything else
		break
	}

	var tmpSizes []Size

	// iterate through all listed sizes
	for _, sku := range prod.VendorProduct.Skus {
		// iterate through all available sizes stored in another map
		for _, availableSku := range prod.VendorProduct.AvailableSkus {
			if availableSku.SkuId == sku.SkuId {

				// iterate through configured sizes attached to product and update state if found in availables
				for _, size := range prod.Sizes {
					if sku.LocalizedSize == size.EuSize {
						size.Available = availableSku.Available
					}
				}

				tmpSizes = append(tmpSizes, Size{
					EuSize:    sku.LocalizedSize,
					Available: availableSku.Available,
				})
			}
		}
	}

	for _, size := range prod.Sizes {
		if size.Available != size.PreviouslyAvailable {
			if size.Available {
				notify(prod, size, true)
			} else {
				notify(prod, size, false)
			}
		}

		// store last state so notification dont repeat
		// and we have the option to notify if sold out
		size.PreviouslyAvailable = size.Available
	}

	var avs []string
	for _, size := range tmpSizes {
		if size.Available {
			avs = append(avs, size.EuSize)
		}
	}

	log.Printf("[%s] found sizes: %s\n", prod.VendorProduct.Title, strings.Join(avs, ", "))
}

func downloadFile(filepath string, url string) (*os.File, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	defer func(Body io.ReadCloser) {
		if err := Body.Close(); err != nil {
			log.Println("error closing reader")
			log.Println(err)
		}
	}(resp.Body)

	out, err := os.Create(filepath)
	if err != nil {
		return nil, err
	}

	defer func(out *os.File) {
		if err := out.Close(); err != nil {
			log.Println("error closing file")
			log.Println(err)
		}
	}(out)

	_, err = io.Copy(out, resp.Body)

	file, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}

	return file, err
}

type Message struct {
	Title        string
	Body         string
	Url          string
	IncludeImage bool
}

func notify(prod *Product, size *Size, up bool) {
	var msg Message

	if up {
		msg = Message{
			Title:        fmt.Sprintf("⚠️ %s", prod.VendorProduct.Title),
			Body:         fmt.Sprintf("Größe %s jetzt verfügbar", size.EuSize),
			Url:          os.Getenv("NIKE_URL"),
			IncludeImage: true,
		}
	} else {
		msg = Message{
			Title: fmt.Sprintf("%s ausverkauft 🙄", prod.VendorProduct.Title),
			Body:  fmt.Sprintf("Größe %s nicht mehr verfügbar", size.EuSize),
			Url:   os.Getenv("NIKE_URL"),
		}
	}

	log.Println(strings.Repeat("#", 120))
	log.Println(strings.Repeat("#", 120))
	log.Println("")
	log.Printf("  %s\n", msg.Title)
	log.Printf("  %s\n", msg.Body)
	log.Println("")
	log.Println(strings.Repeat("#", 120))
	log.Println(strings.Repeat("#", 120))

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

		if msg.IncludeImage {
			thumb := fmt.Sprintf("%s.png", prod.VendorProduct.Id)
			file, err := os.Open(thumb)

			if err != nil && prod.VendorProduct.FirstImageUrl != "" {
				file, err = downloadFile(thumb, prod.VendorProduct.FirstImageUrl)
			}

			if err := message.AddAttachment(file); err != nil {
				log.Println("error attaching pushover file")
				log.Println(err)
			}
		}

		if _, err := app.SendMessage(message, recipient); err != nil {
			log.Println("error sending pushover notification")
			log.Println(err)
		}
	}()
}
