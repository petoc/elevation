package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/julienschmidt/httprouter"
	"github.com/petoc/hgt"
)

type (
	// Location ...
	Location struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	}
	// ResultItem ...
	ResultItem struct {
		Location
		Elevation  float64 `json:"elevation,omitempty"`
		Resolution float64 `json:"resolution,omitempty"`
		Error      int     `json:"error,omitempty"`
	}
	// Response ...
	Response struct {
		Result []*ResultItem `json:"result"`
	}
	// Request ...
	Request struct {
		Locations []*Location `json:"locations"`
	}
)

func locatationNotFound(response *Response, location *Location) {
	response.Result = append(response.Result, &ResultItem{
		Location: *location,
		Error:    404,
	})
}

func jsonLocationHandler(hgtDataDir *hgt.DataDir) func(http.ResponseWriter, *http.Request, httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		locations := []*Location{}
		if r.Method == "GET" {
			query := r.URL.Query()
			queryLocations := query.Get("locations")
			if queryLocations == "" {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			strLocations := strings.Split(queryLocations, "|")
			if len(strLocations) == 1 && strings.Index(strLocations[0], ",") < 1 {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			for _, strLocation := range strLocations {
				coordinates := strings.Split(strLocation, ",")
				location := &Location{}
				if len(coordinates) == 2 {
					location.Latitude, _ = strconv.ParseFloat(coordinates[0], 64)
					location.Longitude, _ = strconv.ParseFloat(coordinates[1], 64)
				}
				locations = append(locations, location)
			}
		} else if r.Method == "POST" {
			if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			request := &Request{}
			err := json.NewDecoder(r.Body).Decode(request)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			locations = request.Locations
		}
		response := &Response{
			Result: []*ResultItem{},
		}
		noLocationFound := true
		for _, location := range locations {
			elevation, resolution, err := hgtDataDir.ElevationAt(location.Latitude, location.Longitude)
			if err != nil {
				locatationNotFound(response, location)
				continue
			}
			noLocationFound = false
			response.Result = append(response.Result, &ResultItem{
				Location:   *location,
				Elevation:  float64(elevation),
				Resolution: float64(resolution),
			})
		}
		if noLocationFound {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		b, err := json.Marshal(response)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(200)
		w.Write(b)
	}
}

func main() {
	var host, dataDir string
	var port int
	flag.StringVar(&host, "host", "127.0.0.1", "Host")
	flag.IntVar(&port, "port", 8080, "Port")
	flag.StringVar(&dataDir, "dir", "data", "Data directory")
	flag.Parse()
	// log.SetFlags(0)
	lruCache, err := lru.NewWithEvict(1000, func(key, value interface{}) {
		if file, ok := value.(*hgt.File); ok {
			file.Close()
		}
	})
	if err != nil {
		log.Fatal(err)
	}
	cache := &hgt.Cache{
		OnGet: func(key string) (*hgt.File, bool) {
			if v, ok := lruCache.Get(key); ok {
				return v.(*hgt.File), true
			}
			return nil, false
		},
		OnAdd: func(key string, value *hgt.File) {
			lruCache.Add(key, value)
		},
		OnClear: func() error {
			lruCache.Purge()
			return nil
		},
	}
	hgtDataDir, err := hgt.OpenDataDir(dataDir, &hgt.DataDirOptions{
		Cache:          cache,
		RangeValidator: hgt.DefaultRangeValidator(),
	})
	if err != nil {
		log.Fatal(err)
	}
	defer hgtDataDir.Close()
	router := httprouter.New()
	router.GlobalOPTIONS = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Access-Control-Request-Method") != "" {
			header := w.Header()
			header.Set("Access-Control-Allow-Methods", r.Header.Get("Allow"))
			header.Set("Access-Control-Allow-Origin", "*")
		}
		w.WriteHeader(http.StatusNoContent)
	})
	router.PanicHandler = func(w http.ResponseWriter, r *http.Request, p interface{}) {
		log.Println(p)
		w.WriteHeader(http.StatusInternalServerError)
	}
	jsonHandler := jsonLocationHandler(hgtDataDir)
	router.GET("/json", jsonHandler)
	router.POST("/json", jsonHandler)
	addr := strings.Join([]string{host, strconv.Itoa(port)}, ":")
	server := &http.Server{
		Addr:         addr,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
		Handler:      router,
	}
	log.Printf("listening=%s", addr)
	log.Fatal(server.ListenAndServe())
}
