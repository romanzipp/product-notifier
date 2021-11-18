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
	"strconv"
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

type Message struct {
	Title        string
	Body         string
	Url          string
	IncludeImage bool
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
			Image: prod.Image,
		}

		for _, prov := range prod.Providers {
			provider, err := GetProviderById(prov.Id, prov.Url)
			if err != nil {
				log.Fatalf("unknown provider: %s\n", prov.Id)
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
			sizes, err := av.Provider.GetAvailableSizes()
			if err != nil {
				av.Log(fmt.Sprintf("error: %s", err.Error()))
			} else if len(sizes) == 0 {
				av.Log("no sizes")
			} else {
				av.Log(fmt.Sprintf("available sizes: %s", strings.Join(sizes, ", ")))

				for _, size := range av.Sizes {
					for _, avSize := range sizes {
						if avSize == size.EuSize {
							size.Available = true
						}
					}

					if size.Available != size.PreviouslyAvailable {
						av.Log(fmt.Sprintf("size %s :: availability %s -> %s", size.EuSize, strconv.FormatBool(size.PreviouslyAvailable), strconv.FormatBool(size.Available)))

						if size.Available {
							av.notify(Message{
								Title:        fmt.Sprintf("⚠️ %s", av.Product.Title),
								Body:         fmt.Sprintf("Size %s NOW AVAILABLE on %s", size.GetEuSize(), av.Provider.GetId()),
								Url:          av.Provider.GetUrl(),
								IncludeImage: true,
							})
						} else {
							av.notify(Message{
								Title: fmt.Sprintf("%s sold out 🙄", av.Product.Title),
								Body:  fmt.Sprintf("Size %s sold out", size.GetEuSize()),
								Url:   av.Provider.GetUrl(),
							})
						}

						size.PreviouslyAvailable = size.Available
					}
				}
			}
		}

		fmt.Println("")
		time.Sleep(time.Duration(cfg.LoopInterval) * time.Second)
	}
}

func (av Availability) notify(msg Message) {
	av.Log(strings.Repeat("#", 120))
	av.Log(fmt.Sprintf("## %s", msg.Title))
	av.Log(fmt.Sprintf("## %s", msg.Body))
	av.Log(strings.Repeat("#", 120))

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

		if msg.IncludeImage && av.Product.Image != "" {
			thumb := fmt.Sprintf("%s.png", av.Product.Title)

			file, err := os.Open(thumb)
			if err != nil {
				file, err = downloadFile(thumb, av.Product.Image)
				if err != nil {
					log.Println("error downloading thumbnail")
					log.Println(err)
				}
			}

			if file != nil {
				if err = message.AddAttachment(file); err != nil {
					log.Println("error attaching pushover file")
					log.Println(err)
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
