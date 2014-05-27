package main

import "./cooldns"

func main() {
	config := cooldns.LoadConfig()
	config.DbFile = "cool.db"
	if config.Domain == "" {
		config.SetDomain("ist.nicht.cool.")
	}
	cooldns.Run(config)
}
