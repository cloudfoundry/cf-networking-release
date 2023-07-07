package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
)

type QueueHandler struct {
	Ch *amqp.Channel
	sync.RWMutex
}

type order struct {
	id                 int            `json:"id,omitempty"`
	customerIdentifier string         `json:"customerIdentifier,omitempty"`
	productOrders      []productOrder `json:"productOrders,omitempty"`
}

type productOrder struct {
	productID string `json:"productId,omitempty"`
	quantity  int    `json:"quantity,omitempty"`
}

func (h *QueueHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	// logger := log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime)
	go func() { h.insertProductForever() }()
	resp.Write([]byte("Starting background process to add orders!"))
}

func (h *QueueHandler) insertProductForever() {
	var err error
	counter := 0

	for j := 0; j < 100_000; j++ {
		counter++
		err = h.insert(makeOrder(counter))
		if err != nil {
			break
		}
	}
}

func makeOrder(count int) order {
	return order{
		id:                 count,
		customerIdentifier: "meow",
		productOrders: []productOrder{
			{productID: "sku9999", quantity: 1},
		},
	}
}

func (h *QueueHandler) insert(o order) error {
	b, err := json.Marshal(o)

	if err != nil {
		return err
	}
	eName, ok := os.LookupEnv("EXCHANGE")
	if !ok {
		eName = "retail.customer.orders"
	}

	err = h.Ch.Publish(
		eName, // exchange
		"",    // routing key
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        b,
		})

	return err
}
