// Command server is the BrewMe API entrypoint.
//
// Responsibility: load config, open the database, build the router, and start
// the HTTP server. Wiring only — no business logic lives here.
package main

import (
	"fmt"
	"net/http"

	"github.com/joho/godotenv"

	"brewme/internal/database"
	"brewme/internal/router"
)

func main() {
	// Load .env BEFORE anything reads the environment, so REDIS_* / DATABASE_DSN
	// from the file (working dir, then backend root) are visible to InitRedis()
	// and Open(). Missing file is not fatal.
	if err := godotenv.Load(); err != nil {
		_ = godotenv.Load("../../.env")
	}

	database.InitRedis()

	if err := database.Open(); err != nil {
		fmt.Println("Database connection failed:", err)
		return
	}
	defer database.DB.Close()

	r := router.Router()

	fmt.Println("Server starts at 8080 port")
	err := http.ListenAndServe(":8080", r)
	if err != nil {
		fmt.Println("Server crushed")
	} else {
		fmt.Println("Server Started")
	}
}
