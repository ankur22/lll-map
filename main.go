package main

import (
	"fmt"
	"net/http"
	"strings"

	"golang.org/x/net/html"
)

func isTitle(t html.Token) (ok bool) {
	for _, a := range t.Attr {
		if strings.Contains(a.Val, "episodeGuide-episodeTitle") {
			ok = true
		}
	}
	return
}

func isSummary(t html.Token) (ok bool) {
	for _, a := range t.Attr {
		if strings.Contains(a.Val, "episodeGuide-episodeSummary") {
			ok = true
		}
	}
	return
}

func getHref(t html.Token) (ok bool, href string) {
	// Iterate over token attributes until we find an "href"
	found := false
	for _, a := range t.Attr {
		if strings.Contains(a.Val, "episodeGuide-episodeLink") {
			found = true
		}
		if a.Key == "href" {
			href = a.Val
			ok = true
		}
	}

	if !found {
		href = ""
		ok = false
	}

	// "bare" return will return the variables (ok, href) as
	// defined in the function definition
	return
}

func crawl(url string, ch chan string, chFinished chan bool) {
	resp, err := http.Get(url)

	defer func() {
		// Notify that we're done after this function
		chFinished <- true
	}()

	if err != nil {
		fmt.Println("ERROR: Failed to crawl:", url)
		return
	}

	b := resp.Body
	defer b.Close() // close Body when the function completes

	z := html.NewTokenizer(b)

	foundTitle := false
	foundSummary := false
	for {
		tt := z.Next()

		switch {
		case tt == html.ErrorToken:
			// End of the document, we're done
			return
		case tt == html.StartTagToken:
			t := z.Token()

			if t.Data == "h3" && isTitle(t) {
				foundTitle = true
				continue
			}
			foundTitle = false

			if t.Data == "p" && isSummary(t) {
				foundSummary = true
				continue
			}
			foundSummary = false

			if t.Data == "a" {
				ok, href := getHref(t)
				if ok {
					ch <- fmt.Sprintf("Link   : %s", href)
				}
			}
		case tt == html.TextToken:
			t := z.Token()
			// if strings.Contains(t.Data, "\n") {
			// 	continue
			// }

			if foundTitle {
				ch <- fmt.Sprintf("Title  : %s", t.Data)
			}

			if foundSummary {
				ch <- fmt.Sprintf("Summary: %s", t.Data)
			}
		}
	}
}

func main() {
	foundUrls := []string{}
	seedUrls := []string{"https://www.channel4.com/programmes/location-location-location/episode-guide/"}

	// Channels
	chUrls := make(chan string)
	chFinished := make(chan bool)

	// Kick off the crawl process (concurrently)
	for _, url := range seedUrls {
		go crawl(url, chUrls, chFinished)
	}

	// Subscribe to both channels
	for c := 0; c < len(seedUrls); {
		select {
		case url := <-chUrls:
			foundUrls = append(foundUrls, url)
		case <-chFinished:
			c++
		}
	}

	// We're done! Print the results...

	fmt.Println("\nFound", len(foundUrls), "unique urls:\n")

	for _, url := range foundUrls {
		fmt.Println(" - " + url)
	}

	close(chUrls)
}
