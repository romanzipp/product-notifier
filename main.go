package main

import (
	"fmt"
	gosxnotifier "github.com/deckarep/gosx-notifier"
	"github.com/gregdel/pushover"
	"github.com/joho/godotenv"
	"github.com/romanzipp/nike-go/config"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type Product struct {
	Title               string
	Image               string
	Size                Size
	Provider            IProvider
	Available           bool
	PreviouslyAvailable bool
}

type Size struct {
	EuSize string
}

func (size Size) GetEuSize() string {
	return size.EuSize
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	cfg := config.ReadConfig()

	var products []*Product

	for _, prod := range cfg.Products {
		for _, size := range prod.Sizes {
			for _, prov := range prod.Providers {
				var provider IProvider

				switch prov.Id {
				case PROVIDER_NIKE:
					provider = Nike{
						Provider{
							Id:  PROVIDER_NIKE,
							Url: prov.Url,
						},
					}
				}

				products = append(products, &Product{
					Title:    prod.Title,
					Image:    prod.Image,
					Size:     Size{size},
					Provider: provider,
				})
			}
		}
	}

	log.Printf("looping in %d seconds\n", cfg.LoopInterval)

	for {
		for _, prod := range products {
			prod.Provider.Check(prod)
		}

		time.Sleep(time.Duration(cfg.LoopInterval) * time.Second)
	}
}

type Message struct {
	Title        string
	Body         string
	Url          string
	IncludeImage bool
}

func (prod Product) Log(line string) {
	log.Printf("[%s] %s :: size %s :: %s", prod.Provider.GetId(), prod.Title, prod.Size.GetEuSize(), line)
}

func (prod Product) notify(up bool) {
	var msg Message

	if up {
		msg = Message{
			Title:        fmt.Sprintf("⚠️ %s", prod.Title),
			Body:         fmt.Sprintf("Größe %s jetzt verfügbar", prod.Size.GetEuSize()),
			Url:          os.Getenv("NIKE_URL"),
			IncludeImage: true,
		}
	} else {
		msg = Message{
			Title: fmt.Sprintf("%s ausverkauft 🙄", prod.Title),
			Body:  fmt.Sprintf("Größe %s nicht mehr verfügbar", prod.Size.GetEuSize()),
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
		note.Group = "com.provider_nike.go"
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
			thumb := fmt.Sprintf("%s.png", prod.Title)
			file, err := os.Open(thumb)

			if err != nil && prod.Image != "" {
				file, err = downloadFile(thumb, prod.Image)
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
