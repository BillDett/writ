package main

import (
	"errors"
	"flag"
	"os"
	"writ/internal/data"
	"writ/internal/ui"
)

/*

writ - a simple multi-document word processor for draft writing

*/

func main() {

	filepath_flag := flag.String("file", "writ.db", "Document file")
	flag.Parse()

	store := data.NewSQLStore()

	// If the data file exists, open it, otherwise create it from scratch
	_, err := os.Stat(*filepath_flag)
	if errors.Is(err, os.ErrNotExist) {
		store.Create(*filepath_flag)
	} else {
		store.Open(*filepath_flag)
	}

	app := ui.NewMainWindow(store)
	app.Init()

	if err := app.Run(); err != nil {
		panic(err)
	}

}
