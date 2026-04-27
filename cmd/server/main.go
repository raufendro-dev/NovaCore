package main

import (
	"log"

	"github.com/raufendro/novacore/internal/app"
)

func main() {
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
