package main

import (
	"os"
	"time"

	"github.com/pivotal-golang/lager"
)

func main() {
	for {
		logger := lager.NewLogger("netman-agent")
		logger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.DEBUG))

		logger.Info("hello")
		time.Sleep(2000 * time.Millisecond)
	}
}
