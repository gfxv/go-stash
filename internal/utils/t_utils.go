package utils

// Utils for tests

import (
	"log"
	"os"
)

func CreateParent(path string) {
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		log.Fatal(err)
	}
}

func CleanUp(path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return
	}

	err := os.RemoveAll(path)
	if err != nil {
		log.Fatal(err)
	}
}
