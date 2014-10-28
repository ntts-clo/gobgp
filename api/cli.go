package main

import (
	"log"
	"github.com/peterh/liner"
)

func main() {
	line := liner.NewLiner()
	defer line.Close()

	for {
		if cmd, err := line.Prompt("shell> "); err != nil {
			log.Print("Error reading line: ", err)
		} else {
			log.Print("Got: ", cmd)
			if cmd == "exit" {
				break
			}
		}
	}
}
