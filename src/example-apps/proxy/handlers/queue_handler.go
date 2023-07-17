package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
)

type QueueHandler struct {
	Ch *amqp.Channel
	Q  amqp.Queue
	sync.RWMutex
}

type message struct {
	Note string
}

func (h *QueueHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	logger := log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime)
	fmt.Println("👉")
	switch req.Method {
	case http.MethodPost:
		var m message

		err := json.NewDecoder(req.Body).Decode(&m)
		fmt.Printf("printf of m: %#v", m)
		if err != nil {
			logger.Println("Body failed decoding")
			http.Error(resp, err.Error(), http.StatusBadRequest)
			return
		}
		fmt.Println("🎉")
		err = h.insert(m.Note)
		if err != nil {
			logger.Println("Failed adding message to exchange")
			http.Error(resp, err.Error(), http.StatusBadRequest)
		}

		resp.WriteHeader(http.StatusOK)
	case http.MethodGet:
		msg, err := h.read()
		if err != nil {
			logger.Println("Failed reading message from queue")
			http.Error(resp, err.Error(), http.StatusBadRequest)
		} else if len(msg) == 0 {
			resp.WriteHeader(http.StatusOK)
			resp.Write([]byte("There were no messages in the queue"))
		} else {
			resp.WriteHeader(http.StatusOK)
			resp.Write(msg)
		}
	}
}

// func (h *QueueHandler) listAll() ([]todo, error) {
// 	rows, err := h.Db.Query("select done, note from todos")

// 	if err != nil {
// 		return []todo{}, err
// 	}

// 	todos := []todo{}
// 	for rows.Next() {
// 		var done bool
// 		var note string
// 		rows.Scan(&done, &note)
// 		fmt.Println("done: ", done)
// 		fmt.Println("note: ", note)
// 		todos = append(todos, todo{Done: done, Note: note})
// 	}

// 	return todos, nil
// }

func (h *QueueHandler) insert(note string) error {
	err := h.Ch.Publish(
		"",       // exchange
		h.Q.Name, // routing key
		false,    // mandatory
		false,    // immediate
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(note),
		})

	return err
}

func (h *QueueHandler) read() ([]byte, error) {
	msg, ok, err := h.Ch.Get(h.Q.Name, true)

	if ok && (err == nil) {
		fmt.Println("🤞 found message!")
		log.Printf(" [x] %s", msg.Body)
		return msg.Body, err
	}

	return []byte{}, err
}
