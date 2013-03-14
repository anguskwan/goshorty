package main

import (
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	"math"
	"net/http"
	"time"
)

type Settings struct {
	RedisUrl       string
	RedisPrefix    string
	RestrictDomain string
	UrlLength      int
}

func AddHandler(resp http.ResponseWriter, req *http.Request) {
	url, err := NewUrl(req.FormValue("url"))
	if err != nil {
		Render(resp, req, "home", map[string]string{"error": err.Error()})
		return
	}

	statsUrl, err := router.Get("stats").URL("id", url.Id)
	if err != nil {
		RenderError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(resp, req, statsUrl.String(), http.StatusFound)
}

func RedirectHandler(resp http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	url, err := GetUrl(vars["id"])
	if err != nil {
		RenderError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	} else if url == nil {
		RenderError(resp, req, "No URL was found with that goshorty code", http.StatusNotFound)
		return
	}

	go url.Hit()
	http.Redirect(resp, req, url.Destination, http.StatusMovedPermanently)
}

func StatsHandler(resp http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	url, err := GetUrl(vars["id"])
	if err != nil {
		RenderError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	} else if url == nil {
		RenderError(resp, req, "No URL was found with that goshorty code", http.StatusNotFound)
		return
	}

	stats, _ := url.Stats()
	Render(resp, req, "stats", map[string]string{
		"id":   url.Id,
		"url":  url.Destination,
		"when": relativeTime(time.Now().Sub(url.Created)),
		"hits": fmt.Sprintf("%d", stats["hits"]),
	})
}

func HomeHandler(resp http.ResponseWriter, req *http.Request) {
	Render(resp, req, "home", nil)
}

func relativeTime(duration time.Duration) string {
	hours := int64(math.Abs(duration.Hours()))
	minutes := int64(math.Abs(duration.Minutes()))
	when := ""
	switch {
	case hours >= (365 * 24):
		when = "Over an year ago"
	case hours > (30 * 24):
		when = fmt.Sprintf("%d months ago", int64(hours/(30*24)))
	case hours == (30 * 24):
		when = "a month ago"
	case hours > 24:
		when = fmt.Sprintf("%d days ago", int64(hours/24))
	case hours == 24:
		when = "yesterday"
	case hours >= 2:
		when = fmt.Sprintf("%d hours ago", hours)
	case hours > 1:
		when = "over an hour ago"
	case hours == 1:
		when = "an hour ago"
	case minutes >= 2:
		when = fmt.Sprintf("%d minutes ago", minutes)
	case minutes > 1:
		when = "a minute ago"
	default:
		when = "just now"
	}
	return when
}

var (
	router = mux.NewRouter()
	settings = new(Settings)
)

func main() {
	var (
		redisHost string
		redisPort int
		redisPrefix string
		regex string
		port int
	)

	flag.StringVar(&redisHost, "redis_host", "", "Redis host (leave empty for localhost)")
	flag.IntVar(&redisPort, "redis_port", 6379, "Redis port")
	flag.StringVar(&redisPrefix, "redis_prefix", "goshorty:", "Redis prefix to use")
	flag.StringVar(&settings.RestrictDomain, "domain", "", "Restrict destination URLs to a single domain")
	flag.IntVar(&settings.UrlLength, "length", 5, "How many characters should the short code have")
	flag.StringVar(&regex, "regex", "[A-Za-z0-9]{%d}", "Regular expression to match route for accessing a short code. %d is replaced with <length> setting")
	flag.IntVar(&port, "port", 8080, "Port where server is listening on")

	flag.Parse()

	regex = fmt.Sprintf(regex, settings.UrlLength)
	settings.RedisUrl = fmt.Sprintf("%s:%d", redisHost, redisPort)
	settings.RedisPrefix = redisPrefix

	router.HandleFunc("/add", AddHandler).Methods("POST").Name("add")
	router.HandleFunc("/{id:"+regex+"}+", StatsHandler).Name("stats")
	router.HandleFunc("/{id:"+regex+"}", RedirectHandler).Name("redirect")
	router.HandleFunc("/", HomeHandler).Name("home")
	for _, dir := range []string{"css", "js", "img"} {
		router.PathPrefix("/" + dir + "/").Handler(http.StripPrefix("/"+dir+"/", http.FileServer(http.Dir("assets/"+dir))))
	}

	fmt.Println(fmt.Sprintf("Server is listening on port %d", port))
	http.ListenAndServe(fmt.Sprintf(":%d", port), router)
}
