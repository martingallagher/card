package main

import (
	"encoding/json"
	"flag"
	"io"
	"os"
	"sync"

	"github.com/martingallagher/card"
)

var (
	dbFile   string
	dbFileMu = &sync.Mutex{}
)

func init() {
	flag.StringVar(&dbFile, "d", "./db.json", "JSON database")
}

func loadDB(filename string) ([]*card.Account, map[int]*card.Account, error) {
	dbFileMu.Lock()

	defer dbFileMu.Unlock()

	f, err := os.Open(filename)

	if os.IsNotExist(err) {
		f, err = os.Create(filename)

		if err != nil {
			return nil, nil, err
		}

		return nil, map[int]*card.Account{}, nil
	} else if err != nil {
		return nil, nil, err
	}

	var accounts []*card.Account

	err = json.NewDecoder(f).Decode(&accounts)

	if err == io.EOF {
		// Assume empty database file
		return nil, map[int]*card.Account{}, nil
	} else if err != nil {
		return nil, nil, err
	}

	accountsMap := make(map[int]*card.Account, len(accounts))

	for _, v := range accounts {
		accountsMap[v.ID] = v
	}

	return accounts, accountsMap, nil
}

func writeDB(filename string, i interface{}) error {
	dbFileMu.Lock()

	defer dbFileMu.Unlock()

	f, err := os.OpenFile(filename, os.O_TRUNC|os.O_WRONLY, os.ModeAppend)

	if err != nil {
		return err
	}

	defer f.Close()

	return json.NewEncoder(f).Encode(i)
}
