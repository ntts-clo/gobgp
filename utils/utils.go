package utils

import (
	"log"
	"os/exec"
)

var REST_PORT = "3000"

func Gen_uuid() string {
	out, err := exec.Command("uuidgen").Output()
	if err != nil {
		log.Fatal(err)
	}
	uuid := string(out[:len(out)-1])
	return uuid
}
