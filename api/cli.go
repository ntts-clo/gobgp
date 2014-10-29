package api

import (
	"github.com/peterh/liner"
	"log"
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
