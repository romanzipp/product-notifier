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
	Id            string `json:"id"`
	Brand         string `json:"brand"`
	Color         string `json:"colorDescription"`
	Title         string `json:"title"`
	FullTitle     string `json:"fullTitle"`
	FirstImageUrl string `json:"firstImageUrl"`
	Skus          []struct {
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

type Content struct {
	Threads struct {
		Products map[string]Product `json:"products"`
	} `json:"Threads"`
}

type Size struct {
	Id        string
	NikeSize  string
	EuSize    string
	Available bool
}

func notify(prod Product, size Size) {
	msg := struct {
		Title string
		Body  string
		Url   string
	}{
		Title: fmt.Sprintf("%s VERFÜGBAR 👟", prod.Title),
		Body:  fmt.Sprintf("Größe %s jetzt verfügbar", size.EuSize),
		Url:   os.Getenv("NIKE_URL"),
	}

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

		thumb := fmt.Sprintf("%s.png", prod.Id)

		file, err := os.Open(thumb)

		if err != nil && prod.FirstImageUrl != "" {
			file, err = downloadFile(thumb, prod.FirstImageUrl)
		}

		message := pushover.NewMessage(msg.Body)
		message.Title = msg.Title
		message.URL = msg.Url

		if err := message.AddAttachment(file); err != nil {
			log.Println("error attaching pushover file")
			log.Println(err)
		}

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

	urls := strings.Split(os.Getenv("NIKE_URLS"), ",")

	interval, _ := strconv.ParseInt(os.Getenv("INTERVAL"), 10, 0)
	fmt.Printf("looping in %d seconds\n", interval)

	sizes := strings.Split(os.Getenv("SIZES"), ",")
	fmt.Printf("searching for sizes %s\n", strings.Join(sizes, ", "))

	for {
		for _, url := range urls {
			check(url, sizes)
		}

		time.Sleep(time.Duration(interval) * time.Second)
	}
}

func check(url string, csizes []string) {
	search := make(map[string]bool)

	for _, csize := range csizes {
		search[csize] = true
	}

	client := http.Client{
		Timeout: time.Second * 5,
	}

	req, _ := http.NewRequest(http.MethodGet, url, nil)

	res, err := client.Do(req)
	if err != nil {
		log.Println("error sending request")
		log.Println(err)
		return
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

	for _, prod := range data.Threads.Products {

		for _, sku := range prod.Skus {

			tsize := Size{
				Id:       sku.SkuId,
				NikeSize: sku.NikeSize,
				EuSize:   sku.LocalizedSize,
			}

			for _, asku := range prod.AvailableSkus {
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
				notify(prod, size)
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

			fmt.Printf("[%s] out of stock... (%s)\n", prod.Title, strings.Join(avs, ", "))
		}
	}
}

func downloadFile(filepath string, url string) (*os.File, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	out, err := os.Create(filepath)
	if err != nil {
		return nil, err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)

	file, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}

	return file, err
}
