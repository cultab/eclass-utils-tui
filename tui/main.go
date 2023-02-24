package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/Huray-hub/eclass-utils/assignments/config"
)

var configuration_options *config.Options

func init() {
	homeCache, err := os.UserCacheDir()
	if err != nil {
		log.Fatal(err.Error())
	}

	path := filepath.Join(homeCache, "eclass-tui")
	if _, err = os.Stat(path); errors.Is(err, os.ErrNotExist) {
		err = os.MkdirAll(path, 0755)
		if err != nil {
			log.Fatal(err)
		}
	}

	file, err := os.OpenFile(
		filepath.Join(path, "assignments.log"),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		0644,
	)
	if err != nil {
		log.Fatal(err)
	}
    // defer func() { NOTE: uhhhhhhhh, remove? ask
	// 	if err = file.Close(); err != nil {
	// 		log.Fatal(err.Error())
	// 	}
	// }()

	log.SetOutput(file)

	configuration_options, _, err = config.Import()
	if err != nil {
		log.Fatal(err)
	}
}

func updateConfiguration(o config.Options) {
    log.Print("Not Implemented: updating the configuration file.")
}

func main() {

    m := NewList()
	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}

}
