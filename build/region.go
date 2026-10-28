package main

// region.go — what differs between the ru and com deployments apart from env
// config: the zone's domain and the language of every user-facing string. The
// values live in region-ru.go and region-com.go. Technical text (logs,
// "bad request"-style errors) is English in both zones and stays out of here.
// Infrastructure URLs (auth-center) stay in env.

import (
	"fmt"
	"log"
	"os"
	"reflect"
)

type regionDef struct {
	Domain string // sh-development.<zone>: the login link and the menu redirect
	Lang   string // <html lang>

	Text pageText
}

// pageText holds the strings of web/index.html. Fields, not a map: a typo in
// the template fails at render instead of silently printing nothing.
type pageText struct {
	// login screen
	Login string // login button
	About string // text under the login card

	// header
	Tab1   string
	Tab2   string
	Tab3   string
	Apps   string // profile menu
	Readme string
	Logout string

	JS jsText // strings set by script.js
}

// jsText is rendered into the page as JSON (window.TEXT) and read by script.js.
type jsText struct {
	Error string `json:"error"` // a failed request without a server message
}

var regions = map[string]*regionDef{
	"ru":  &regionRU,
	"com": &regionCOM,
}

// Active region, set once by initRegion before anything serves.
var (
	regionName string
	region     *regionDef
	domain     string // region.Domain; auth-server.go builds the menu redirect from it
)

// initRegion picks the region by REGION (ru | com, set by the deploy matrix).
// Empty means local dev and falls back to ru; anything else is a config error
// and stops the process rather than serving the wrong zone — the health check
// then fails and deploy.sh rolls back. A string forgotten in either region is
// fatal too, whichever region this instance serves.
func initRegion() {
	for name, r := range regions {
		if err := checkFilled(reflect.ValueOf(*r), "region"); err != nil {
			log.Fatalf("region %s: %v", name, err)
		}
	}

	regionName = os.Getenv("REGION")
	if regionName == "" {
		regionName = "ru"
	}
	r, ok := regions[regionName]
	if !ok {
		log.Fatalf("unknown REGION %q (want ru or com)", regionName)
	}
	region = r
	domain = r.Domain
	log.Printf("region name=%s domain=%s lang=%s", regionName, domain, region.Lang)
}

// checkFilled walks structs and fails on the first empty string.
func checkFilled(v reflect.Value, path string) error {
	switch v.Kind() {
	case reflect.String:
		if v.Len() == 0 {
			return fmt.Errorf("%s is empty", path)
		}
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if err := checkFilled(v.Field(i), path+"."+v.Type().Field(i).Name); err != nil {
				return err
			}
		}
	}
	return nil
}
