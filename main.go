package main

import "url-shortener/api"

func main() {
	api := api.New()
	api.Mount()
	api.Run()
}
