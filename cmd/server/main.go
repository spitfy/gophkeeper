package main

import (
	"fmt"
	"gophkeeper/internal/config"
	"gophkeeper/internal/utils/logger"
)

func main() {
	conf := config.NewConfig()
	log := logger.NewLogger(conf.Env)
	log.Info("test")
	fmt.Println("Start")
}
