package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

type TodosHandler struct {
	Db *sql.DB
	sync.RWMutex
}

type todo struct {
	done bool
	note string
}

func (h *TodosHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	fmt.Println("👉")
	var t todo
	logger := log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime)

	err := json.NewDecoder(req.Body).Decode(&t)
	if err != nil {
		logger.Println("Body failed decoding")
		http.Error(resp, err.Error(), http.StatusBadRequest)
		return
	}
	fmt.Println("🎉")
	err = h.insert(t)
	if err != nil {
		logger.Println("Failed creating todo")
		http.Error(resp, err.Error(), http.StatusBadRequest)
	}

	resp.WriteHeader(http.StatusOK)
}

func (h *TodosHandler) insert(t todo) error {
	query := "INSERT INTO todos(done, note) VALUES (?, ?)"
	ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
	fmt.Println("✅")
	defer cancelfunc()
	stmt, err := h.Db.PrepareContext(ctx, query)
	if err != nil {
		log.Printf("Error %s when preparing SQL statement", err)
		return err
	}
	defer stmt.Close()

	fmt.Println("🔥")
	res, err := stmt.ExecContext(ctx, t.done, t.note)
	if err != nil {
		log.Printf("Error %s when inserting row into products table", err)
		return err
	}
	rows, err := res.RowsAffected()
	fmt.Println("🥾")
	if err != nil {
		log.Printf("Error %s when finding rows affected", err)
		return err
	}
	log.Printf("%d products created ", rows)
	return nil
}
