package main

import (
	"fmt"

	"github.com/ksvaza/server/master"
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			print(fmt.Sprintf("Panic:%v", r))
		}
	}()

	server := master.NewService("svaza.lv", 1883, "esp32", "T-SIM7000G")
	server.Run()

	print("Hello, world!\n")
}
