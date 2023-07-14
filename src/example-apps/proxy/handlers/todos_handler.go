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
	Done bool
	Note string
}

func (h *TodosHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	logger := log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime)
	fmt.Println("👉")
	switch req.Method {
	case http.MethodPost:
		var t todo

		err := json.NewDecoder(req.Body).Decode(&t)
		fmt.Printf("printf of t: %#v", t)
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
	case http.MethodGet:
		todos, err := h.listAll()
		if err != nil {
			logger.Println("Failed getting todos")
			http.Error(resp, err.Error(), http.StatusInternalServerError)
		}

		respBytes, err := json.Marshal(todos)
		if err != nil {
			logger.Println("Failed marshalling todos")
			http.Error(resp, err.Error(), http.StatusInternalServerError)
		}
		resp.WriteHeader(http.StatusOK)
		resp.Write(respBytes)
	}
}

func (h *TodosHandler) listAll() ([]todo, error) {
	rows, err := h.Db.Query("select done, note from todos")

	if err != nil {
		return []todo{}, err
	}

	todos := []todo{}
	for rows.Next() {
		var done bool
		var note string
		rows.Scan(&done, &note)
		fmt.Println("done: ", done)
		fmt.Println("note: ", note)
		todos = append(todos, todo{Done: done, Note: note})
	}

	return todos, nil
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
	fmt.Println("adding done: ", t.Done)
	fmt.Println("adding note: ", t.Note)
	res, err := stmt.ExecContext(ctx, t.Done, t.Note)
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
