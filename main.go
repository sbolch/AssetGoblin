package main

import (
	"asset-manager/image"
	"asset-manager/settings"
	"net/http"
	"os"
)

type config struct {
	Port string `json:"port"`
}

var options config

func init() {
	settings.Load(&options)
}

func main() {
	wd, _ := os.Getwd()

	http.HandleFunc("/img/", image.Serve)
	http.Handle("/", http.FileServer(http.Dir(wd+"/public")))

	err := http.ListenAndServe(":"+options.Port, nil)
	if err != nil {
		panic(err)
	}
}
