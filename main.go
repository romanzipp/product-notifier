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
	Title string
	Image string
}

type Availability struct {
	Product  *Product
	Sizes    []*Size
	Provider IProvider
}

func (av *Availability) Check() {
	av.Provider.Check(av)
}

func (av *Availability) Log(line string) {
	log.Printf("[%s] %s :: %s", av.Provider.GetId(), av.Product.Title, line)
}

type Size struct {
	EuSize              string
	Available           bool
	PreviouslyAvailable bool
}

func (size Size) GetEuSize() string {
	return size.EuSize
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	cfg := config.ReadConfig()

	var availabilities []*Availability

	for _, prod := range cfg.Products {
		var sizes []*Size
		product := &Product{
			Title: prod.Title,
		}

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
			case PROVIDER_ZALANDO:
				provider = Zalando{
					Provider{
						Id:  PROVIDER_ZALANDO,
						Url: prov.Url,
					},
				}
			default:
				log.Fatalf("unknown provider: %s", prov.Id)
			}

			for _, size := range prod.Sizes {
				sizes = append(sizes, &Size{
					EuSize: size,
				})
			}

			availabilities = append(availabilities, &Availability{
				Product:  product,
				Sizes:    sizes,
				Provider: provider,
			})
		}
	}

	log.Printf("looping in %d seconds\n", cfg.LoopInterval)

	for {
		for _, av := range availabilities {
			av.Check()
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

func (av Availability) notify(size *Size, up bool) {
	var msg Message

	if up {
		msg = Message{
			Title:        fmt.Sprintf("⚠️ %s", av.Product.Title),
			Body:         fmt.Sprintf("Size %s NOW AVAILABLE", size.GetEuSize()),
			Url:          av.Provider.GetUrl(),
			IncludeImage: true,
		}
	} else {
		msg = Message{
			Title: fmt.Sprintf("%s sold out 🙄", av.Product.Title),
			Body:  fmt.Sprintf("Größe %s nicht mehr verfügbar", size.GetEuSize()),
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
			thumb := fmt.Sprintf("%s.png", av.Product.Title)
			file, err := os.Open(thumb)

			if err != nil && av.Product.Image != "" {
				file, err = downloadFile(thumb, av.Product.Image)
				if err != nil {
					if err := message.AddAttachment(file); err != nil {
						log.Println("error attaching pushover file")
						log.Println(err)
					}
				}
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
