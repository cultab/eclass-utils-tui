package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"assignments/config"

	tea "github.com/charmbracelet/bubbletea"
)

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

}

func updateConfiguration(o config.Options) {
    log.Print("Not Implemented: updating the configuration file.")
}

func main() {
    opts, creds, err := config.Import()
	if err != nil {
		log.Fatal(err)
	}
    config := config.Config{
        Credentials: *creds,
        Options: *opts,
    }

    // config.Options.ExcludedCourses[""] = struct{}{}
    m := NewList(config)
	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}

}
