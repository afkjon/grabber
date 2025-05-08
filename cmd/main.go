package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	db "github.com/afkjon/grabber/internal/database"
	model "github.com/afkjon/grabber/internal/model"
	"github.com/gocolly/colly/v2"
	"github.com/urfave/cli/v3"
)

// main is the entry point for the application
func main() {
	cli.VersionFlag = &cli.BoolFlag{
		Name:    "print-version",
		Aliases: []string{"v"},
		Usage:   "print only the version",
	}

	cmd := &cli.Command{
		Commands: []*cli.Command{
			{
				Name:  "clean",
				Usage: "Cleans and autopopulates shop locations based on Google API",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "type",
						Value: "csv",
						Usage: "Type of file to clean",
					},
				},
				Action: func(context.Context, *cli.Command) error {
					fmt.Println("boom")
					return nil
				},
			},
			{
				Name:  "crawl",
				Usage: "Crawls locations",
				/*
					Flags: []cli.Flag{
						&cli.StringFlag{
							Name:    "depth",
							Aliases: []string{"d"},
							Value:   "1",
							Usage:   "Type of file to clean",
						},
					},
				*/
				Action: func(context.Context, *cli.Command) error {
					// Notify channel for specific signals
					sigChan := make(chan os.Signal, 1)
					signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

					err := db.Connect()
					if err != nil {
						fmt.Fprintf(os.Stderr, "Error: %v\n", err)
						os.Exit(1)
					}
					fmt.Println("Crawling...")

					shops := scrapeTabelog("tokyo")
					err = db.InsertShops(shops)
					if err != nil {
						fmt.Fprintf(os.Stderr, "Error: %v\n", err)
						os.Exit(1)
					}
					for _, shop := range shops {
						scrapeAddressFromTabelogPage(shop)
					}

					sig := <-sigChan
					fmt.Printf("\nReceived signal: %v\n", sig)

					return nil
				},
			},
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}

func processJob(ctx context.Context, job any) {
	fmt.Printf("Processing job: %v\n", job)
}

func jobWorker() {
	for {
		jobList, err := db.GetPendingJobs()
		if err != nil {
			log.Fatal(err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		for _, job := range jobList {

			processJob(ctx, job)
		}
	}
}

// scrapeTabelog scrapes Tabelog for shops
func scrapeTabelog(location string) []model.Shop {
	c := colly.NewCollector(
		colly.Async(true),
		colly.MaxDepth(1),
	)

	c.Limit(&colly.LimitRule{DomainGlob: "*", Parallelism: 2})

	var shops []model.Shop

	c.OnHTML("div.list-rst__rst-data", func(e *colly.HTMLElement) {
		areaText := strings.Split(e.ChildText(".list-rst__area-genre"), " ")
		station := areaText[0]
		dist := areaText[1]

		shops = append(shops, model.Shop{
			Name:            e.ChildText("h3.list-rst__rst-name a.list-rst__rst-name-target"),
			TabelogURL:      e.ChildAttr("a.list-rst__rst-name-target", "href"),
			Price:           e.ChildText("span.c-rating-v3__val"),
			Station:         station,
			StationDistance: dist,
			CityId:          1,
		})
	})

	c.OnRequest(func(r *colly.Request) {
		fmt.Println("Visiting", r.URL.String())
	})

	// Callback for when scraping is finished
	c.OnScraped(func(r *colly.Response) {
		fmt.Println("Finished scraping:", r.Request.URL)
	})

	//todo: refactor defaults
	if location == "" {
		location = "tokyo"
	}

	var date = time.Now().Format("20060102")

	url := fmt.Sprintf(`https://tabelog.com/%[1]srstLst/?vs=1&sa=&sk=ramen&lid=top_navi1&vac_net=&svd=%[2]s&svt=2000&svps=2&hfc=1&sw=ramen`, location+"/", date)
	err := c.Visit(url)
	if err != nil {
		log.Fatal(err)
	}
	c.Wait()

	fmt.Println("Found", len(shops), "shops")

	return shops
}

func scrapeAddressFromTabelogPage(shop model.Shop) {
	c := colly.NewCollector(
		colly.Async(true),
		colly.MaxDepth(1),
	)

	c.Limit(&colly.LimitRule{DomainGlob: "*", Parallelism: 2})

	c.OnHTML("p.rstinfo-table__address", func(e *colly.HTMLElement) {
		shop.Address = e.Text
	})

	c.OnRequest(func(r *colly.Request) {
		fmt.Println("Visiting", r.URL.String())
	})

	// Callback for when scraping is finished
	c.OnScraped(func(r *colly.Response) {
		fmt.Println("Finished scraping:", r.Request.URL)
	})

	err := c.Visit(shop.TabelogURL)
	if err != nil {
		log.Fatal(err)
	}
	c.Wait()

	db.UpdateShop(shop)
}
