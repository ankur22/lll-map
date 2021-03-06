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

	return
}

func crawl(url string, ch chan string) {
	defer close(ch)

	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("ERROR: Failed to crawl:", url)
		return
	}

	b := resp.Body
	defer b.Close()

	z := html.NewTokenizer(b)

	foundTitle := false
	foundSummary := false
	for {
		tt := z.Next()

		switch {
		case tt == html.ErrorToken:
			// End of the document
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
					ch <- fmt.Sprintf("Link:    %s", href)
				}
			}
		case tt == html.TextToken:
			t := z.Token()

			if foundTitle {
				ch <- fmt.Sprintf("Title:   %s", t.Data)
			}

			if foundSummary {
				ch <- fmt.Sprintf("Summary: %s", t.Data)
			}
		}
	}
}

type episode struct {
	title, summary, link string
}

func newEpisode(title, summary, link string) episode {
	return episode{
		title:   title,
		summary: summary,
		link:    link,
	}
}

func (e episode) String() string {
	return fmt.Sprintf("Title: %s | Summary: %s | Link: %s", e.title, e.summary, e.link)
}

func main() {
	url := "https://www.channel4.com/programmes/location-location-location/episode-guide/"

	// Channels
	chEpisodeValues := make(chan string)

	// Kick off the crawl process (concurrently)
	go crawl(url, chEpisodeValues)

	// Subscribe to both channels
	var episodes []episode
	var title, summary, link string
	for v := range chEpisodeValues {
		if strings.Contains(v, "Title:") {
			if title != "" {
				episodes = append(episodes, newEpisode(title, summary, link))
			}
			title = strings.Replace(v, "Title:   ", "", 1)
		} else if strings.Contains(v, "Summary:") {
			if summary != "" {
				episodes = append(episodes, newEpisode(title, summary, link))
			}
			summary = strings.Replace(v, "Summary: ", "", 1)
		} else if strings.Contains(v, "Link:") {
			if link != "" {
				episodes = append(episodes, newEpisode(title, summary, link))
			}
			link = strings.Replace(v, "Link:    ", "", 1)
		}
	}

	fmt.Println("\nFound", len(episodes), "episodes:")

	for _, e := range episodes {
		fmt.Println(e)
	}
}
