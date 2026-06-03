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
	// Load .env if present (searches the working directory, then the backend
	// root) so DATABASE_DSN etc. are available. Missing file is not fatal.
	if err := godotenv.Load(); err != nil {
		_ = godotenv.Load("../../.env")
	}

	if err := database.Open(); err != nil {
		fmt.Println("Database connection failed:", err)
		return
	}
	defer database.DB.Close()

	r := router.Router()

	err := http.ListenAndServe(":8080", r)
	if err != nil {
		fmt.Println("Server crushed")
	} else {
		fmt.Println("Server Started")
	}
}
