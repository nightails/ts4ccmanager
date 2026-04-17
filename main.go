package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/andygrunwald/vdf"
)

func main() {
	// Parse the libraryfolders.vdf and display the result
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	lf := filepath.Join(home, ".steam", "steam", "steamapps", "libraryfolders.vdf")

	f, err := os.Open(lf)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	p := vdf.NewParser(f)
	m, err := p.Parse()
	if err != nil {
		log.Fatal(err)
	}

	lmap := m["libraryfolders"].(map[string]interface{})
	lib2 := lmap["2"].(map[string]interface{})
	apps := lib2["apps"].(map[string]interface{})
	_, ok := apps["1222670"]
	if ok {
		fmt.Println("Found Sims 4 on Steam!")
	} else {
		fmt.Println("Sims 4 not found on Steam!")
	}
}
