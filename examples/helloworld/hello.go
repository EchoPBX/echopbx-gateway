package main

import (
	"fmt"
	"time"

	"github.com/EchoPBX/echopbx-gateway/internal/events"
	"github.com/EchoPBX/echopbx-gateway/pkg/sdk"
)

// Init es el entrypoint obligatorio
func Init(bus *events.Bus) {
	fmt.Println("HelloWorld plugin started ✅")

	go func() {
		for {
			bus.Publish(sdk.Event{
				Type: "hello",
				Data: map[string]any{"msg": "Hello from plugin"},
			})
			time.Sleep(5 * time.Second)
		}
	}()
}
