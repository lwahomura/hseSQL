package main

import (
	"hseSQL/internal/runner"
	"log"
)

func main() {
	c, err := runner.ReadConfig("configs/config.yaml")
	if err != nil {
		log.Fatal(err)
	}
	r, err := runner.NewRunner(c)
	if err != nil {
		log.Fatal(err)
	}
	r.Run()
}
