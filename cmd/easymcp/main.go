package main

import (
	"context"
	"log"

	"github.com/example/easymcp/internal/server"
)

func main() {
	if err := server.Run(context.Background(), ":8080", nil, nil); err != nil {
		log.Fatal(err)
	}
}
