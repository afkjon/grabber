package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/afkjon/grabber/internal/crawlers"
	db "github.com/afkjon/grabber/internal/database"
	"github.com/joho/godotenv"
	"github.com/urfave/cli/v3"
)

// main is the entry point for the application
func main() {
	cli.VersionFlag = &cli.BoolFlag{
		Name:    "print-version",
		Aliases: []string{"v"},
		Usage:   "print only the version",
	}

	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file")
		os.Exit(1)
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
				Action: func(ctx context.Context, cmd *cli.Command) error {
					// Notify channel for specific signals
					sigChan := make(chan os.Signal, 1)
					signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

					err := db.Connect()
					if err != nil {
						fmt.Fprintf(os.Stderr, "Error: %v\n", err)
						os.Exit(1)
					}
					defer db.Close()

					var location = cmd.Args().Get(0)
					fmt.Println("Crawling for shops at " + location)

					shops := crawlers.ScrapeTabelog(location)
					err = db.InsertShops(shops)
					if err != nil {
						fmt.Fprintf(os.Stderr, "Error: %v\n", err)
						os.Exit(1)
					}
					for _, shop := range shops {
						crawlers.ScrapeAddressFromTabelogPage(shop)
					}
					//sig := <-sigChan
					//fmt.Printf("\nReceived signal: %v\n", sig)

					return nil
				},
			},
			{
				Name:  "geocode",
				Usage: "Format addresses",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					fmt.Println("Geocoding addresses for shops in database")

					err := crawlers.GeocodeAddresses()
					if err != nil {
						fmt.Fprintf(os.Stderr, "Error: %v\n", err)
						os.Exit(1)
					}

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
