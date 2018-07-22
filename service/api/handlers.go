package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"sync"

	"github.com/cockroachdb/apd"
	"github.com/go-chi/chi"
	"github.com/martingallagher/card"
	"go.uber.org/zap"
)

var (
	accounts    []*card.Account
	accountsMap = map[int]*card.Account{}
	accountsMu  = &sync.RWMutex{}
)

func writeJSON(w http.ResponseWriter, statusCode int, i interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	err := json.NewEncoder(w).Encode(i)

	if err != nil {
		logger.Error("Failed encoding JSON", zap.Error(err))
	}
}

func updateDB(w http.ResponseWriter, i interface{}) {
	err := writeDB(dbFile, accounts)

	if err != nil {
		logger.Error("Failed to write to database", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	writeJSON(w, http.StatusOK, i)
}

func getAccounts(w http.ResponseWriter, r *http.Request) {
	accountsMu.RLock()
	writeJSON(w, http.StatusOK, accounts)
	accountsMu.RUnlock()
}

func createAccount(w http.ResponseWriter, r *http.Request) {
	var newAccount struct {
		ID int `json:"id"`
	}

	err := json.NewDecoder(r.Body).Decode(&newAccount)

	if err != nil {
		logger.Error("Failed to decode JSON", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)

		return
	}

	accountsMu.Lock()

	defer accountsMu.Unlock()

	_, exists := accountsMap[newAccount.ID]

	if exists {
		w.WriteHeader(http.StatusConflict)

		return
	}

	account := card.NewAccount(newAccount.ID)
	accounts = append(accounts, account)
	accountsMap[account.ID] = account

	updateDB(w, account)
}

func getAccountValue(w http.ResponseWriter, r *http.Request) (*card.Account, error) {
	idParam := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idParam)

	if err != nil {
		logger.Error("Invalid account ID", zap.String("id", idParam), zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)

		return nil, err
	}

	account, exists := accountsMap[id]

	if !exists {
		w.WriteHeader(http.StatusNotFound)

		return nil, errors.New("account not found")
	}

	return account, nil
}

func getAccount(w http.ResponseWriter, r *http.Request) {
	account, err := getAccountValue(w, r)

	if err != nil {
		return
	}

	writeJSON(w, http.StatusOK, account)
}

func statement(w http.ResponseWriter, r *http.Request) {
	accountsMu.Lock()

	defer accountsMu.Unlock()

	account, err := getAccountValue(w, r)

	if err != nil {
		return
	}

	statement, err := account.Statement()

	if err != nil {
		logger.Error("Failed to generate statement", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	w.Write([]byte(statement))
}

func load(w http.ResponseWriter, r *http.Request) {
	accountsMu.Lock()

	defer accountsMu.Unlock()

	account, err := getAccountValue(w, r)

	if err != nil {
		return
	}

	var load struct {
		Amount string `json:"amount"`
	}

	err = json.NewDecoder(r.Body).Decode(&load)

	if err != nil {
		logger.Error("Failed to decode JSON", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)

		return
	}

	d, _, err := apd.NewFromString(load.Amount)

	if err != nil {
		logger.Error("Failed to decode load request", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)

		return
	}

	err = account.Load(d)

	if err != nil {
		logger.Error("Failed to load amount", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	updateDB(w, account)
}

func transaction(w http.ResponseWriter, r *http.Request, op card.Operation) {
	accountsMu.Lock()

	defer accountsMu.Unlock()

	account, err := getAccountValue(w, r)

	if err != nil {
		return
	}

	var req struct {
		MerchantID int    `json:"merchantID"`
		Amount     string `json:"amount"`
	}

	err = json.NewDecoder(r.Body).Decode(&req)

	if err != nil {
		logger.Error("Failed to decode JSON", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)

		return
	}

	d, _, err := apd.NewFromString(req.Amount)

	if err != nil {
		logger.Error("Failed to decode request", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)

		return
	}

	switch op {
	case card.Authorize:
		err = account.Authorize(req.MerchantID, d)
	case card.Capture:
		err = account.Capture(req.MerchantID, d)
	case card.Reverse:
		err = account.Reverse(req.MerchantID, d)
	case card.Refund:
		err = account.Refund(req.MerchantID, d)
	default:
		logger.Error("Unknown operation", zap.Uint8("op", uint8(op)))
		w.WriteHeader(http.StatusBadRequest)

		return
	}

	if err != nil {
		logger.Error("Failed to perform request", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	updateDB(w, account)
}

func authorize(w http.ResponseWriter, r *http.Request) {
	transaction(w, r, card.Authorize)
}

func capture(w http.ResponseWriter, r *http.Request) {
	transaction(w, r, card.Capture)
}

func reverse(w http.ResponseWriter, r *http.Request) {
	transaction(w, r, card.Reverse)
}

func refund(w http.ResponseWriter, r *http.Request) {
	transaction(w, r, card.Refund)
}
