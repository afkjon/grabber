package crawlers

import (
	"fmt"
	"log"
	"strings"
	"time"

	db "github.com/afkjon/grabber/internal/database"
	"github.com/afkjon/grabber/internal/model"
	"github.com/gocolly/colly"
)

// ScrapeTabelog scrapes Tabelog for shops
func ScrapeTabelog(location string) []model.Shop {
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

// ScrapeAddressFromTabelogPage scrapes the address from the Tabelog page
// then updates the shop in the database
func ScrapeAddressFromTabelogPage(shop model.Shop) {
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
