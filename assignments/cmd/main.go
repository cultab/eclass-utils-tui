package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"assignments/assignment"
	"assignments/calendar"
	"assignments/cmd/flags"
	"assignments/cmd/output"
	"assignments/config"
)

func init() {
	homeCache, err := os.UserCacheDir()
	if err != nil {
		log.Fatal(err.Error())
	}

	path := filepath.Join(homeCache, "eclass-utils")
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
	defer func() {
		if err = file.Close(); err != nil {
			log.Fatal(err.Error())
		}
	}()

	log.SetOutput(file)
}

func main() {
	opts, creds, err := config.Import()
	if err != nil {
		log.Fatal(err.Error())
	}

	flags.Read(opts, creds)

	err = config.Ensure(opts, creds)
	if err != nil {
		log.Fatal(err.Error())
	}

	a, err := assignment.Get(opts, creds)
	if err != nil {
		log.Fatal(err.Error())
	}

	err = output.PrintAssignments(a, opts.PlainText)
	if err != nil {
		log.Fatal(err.Error())
	}

	if opts.ExportICS {
		path, err := calendar.Export(a, opts.BaseDomain)
		if err != nil {
			log.Fatal(err.Error())
		}

		fmt.Printf("stored in\n%v\n", path)
	}
}
