package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
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
	c.OnHTML("h3.list-rst__rst-name a.list-rst__rst-name-target", func(e *colly.HTMLElement) {
		shop := model.Shop{
			Name:       e.Text,
			TabelogURL: e.Attr("href"),
		}
		//e.Request.Visit(e.Attr("href"))
		shops = append(shops, shop)
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
	fmt.Println(shops)

	return shops
}
