package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"net/http"
	netUrl "net/url"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/Stolkerve.com/miron-search-engine/components"
	"github.com/Stolkerve.com/miron-search-engine/db"
	"github.com/Stolkerve.com/miron-search-engine/models"
	tfidf "github.com/Stolkerve.com/miron-search-engine/tf_idf"
	"github.com/gocolly/colly/v2"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	gommonLog "github.com/labstack/gommon/log"
)

func main() {
	urlCache := make(map[string]bool)

	if err := godotenv.Load(); err != nil {
		log.Fatal(err)
	}
	db.SetupDB()
	e := echo.New()

	// Static
	e.Static("/assets", "assets")

	// Middlewares
	//e.Use(middleware.Logger())
	e.Use(middleware.RecoverWithConfig(middleware.RecoverConfig{
		StackSize: 1 << 10, // 1 KB
		LogLevel:  gommonLog.ERROR,
	}))
	e.Use(middleware.GzipWithConfig(middleware.GzipConfig{
		Level: 5,
	}))

	// Controllers
	e.GET("/", func(c echo.Context) error {
		// User query input
		search := c.QueryParam("search")

		if len(search) != 0 {
			parser := tfidf.Parser{}
			wordsFreqMap, _ := parser.ParseText(search)

			results := make(map[string]float64)
			for w := range wordsFreqMap {
				tfIdfData := make([]models.TfIdfData, 0)

				nDocsTerm := 0
				nDocs := 0
				db.Pool.Raw("SELECT COUNT(word_freqs.count) FROM word_freqs WHERE word_freqs.word = ?", w).Scan(&nDocsTerm)
				db.Pool.Raw("SELECT COUNT(documents.id) FROM documents").Scan(&nDocs)
				if nDocsTerm == 0 {
					nDocsTerm = 1.0
				}
				idf := math.Log10(float64(nDocs) / float64(nDocsTerm))
				db.Pool.Raw(db.TF_IDF_QUERY, idf, w).Scan(&tfIdfData)

				for _, v := range tfIdfData {
					if _, ok := results[v.Url]; ok {
						results[v.Url] += v.TfIdf
						continue
					}
					results[v.Url] = v.TfIdf
				}

			}

			urls := make([]string, len(results))
			i := 0
			for url := range results {
				// fmt.Printf("URL: %v\n\tScore:%v\n.", url, score)
				urls[i] = url
				i += 1
			}

			sort.SliceStable(urls, func(i, j int) bool {
				return results[urls[i]] > results[urls[j]]
			})

			if err := components.Search(urls).Render(context.Background(), c.Response().Writer); err != nil {
				return c.String(http.StatusInternalServerError, "Fail to render the html")
			}
			return nil
		}

		if err := components.Index().Render(context.Background(), c.Response().Writer); err != nil {
			return c.String(http.StatusInternalServerError, "Fail to render the html")
		}
		return nil
	})

	e.POST("/indexing", func(c echo.Context) error {
		url := c.FormValue("url")
		urlVisitorError := false
		urls := make([]string, 0)

		scraperUrlVisitor := colly.NewCollector(colly.MaxDepth(1))
		scraperUrlVisitor.OnError(func(r *colly.Response, err error) {
			urlVisitorError = true
		})
		scraperUrlVisitor.OnHTML("a[href]", func(e *colly.HTMLElement) {
			link := e.Attr("href")
			//fmt.Printf("Link found: %q -> %s\n", e.Text, link)
			absolute := e.Request.AbsoluteURL(link)
			if len(absolute) != 0 {
				url, _ := netUrl.Parse(absolute)
				domains := strings.Split(url.Hostname(), ".")
				if len(domains[0]) <= 3 && domains[0] != "es" && domains[0] != "en" || len(absolute) > 2047 {
					return
				}
				urls = append(urls, absolute)
				e.Request.Visit(absolute)
			}
		})
		scraperUrlVisitor.Visit(url)

		components.Indexingbar(&urlVisitorError).Render(context.Background(), c.Response().Writer)

		scraper := colly.NewCollector(colly.Async(true))
		scraper.WithTransport(&http.Transport{
			DisableKeepAlives: true,
		})
		scraper.OnHTML("body", func(e *colly.HTMLElement) {
			parser := tfidf.Parser{}
			wordsFeqMap, count := parser.ParseText(e.Text)
			wordsFeq := make([]models.WordFreq, len(wordsFeqMap))
			i := 0
			for _, wf := range wordsFeqMap {
				wordsFeq[i] = wf
				i += 1
			}
			doc := models.Document{
				Url:        e.Request.URL.String(),
				Words:      wordsFeq,
				WordsCount: count,
			}
			if err := db.Pool.Create(&doc).Error; err != nil {
				return
			}
			db.Pool.Save(&doc)
		})

		fmt.Printf("Indexando %v nuevos documentos\n", len(urls))

		for i, url := range urls {
			if i > 100 {
				break
			}
			i := i + 1
			if _, ok := urlCache[url]; !ok {
				urlCache[url] = true
				go func(url string, i int) {
					time.Sleep(time.Duration(i*5) * time.Second)
					scraper.Visit(url)
					fmt.Printf("Indexando %v.\n", url)
				}(url, i)
			}

		}

		return nil
	})

	e.Logger.Fatal(e.Start(fmt.Sprintf(":%s", os.Getenv("PORT"))))
}
