package main

import (
	"fmt"

	"github.com/ksvaza/server/master"
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Panic:%v\n", r)
		}
	}()

	server := master.NewService("svaza.lv", 1883, "esp32", "T-SIM7000G", "http://localhost:8086", "lGnMEQI7KFmOaE-IYKvn7aGi7raeKew3-wwT6_9iYTuV2SQBrzPMDUuo46z0AsbM5qeJooRMGyp5ZsouIXeSKw==")
	server.Run()

	print("Hello, world!\n")
}
