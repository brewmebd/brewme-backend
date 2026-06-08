// Command server is the BrewMe API entrypoint.
//
// Responsibility: load config, open the database, build the router, and start
// the HTTP server. Wiring only — no business logic lives here.
package main

import (
	"fmt"
	"net/http"
	"os"

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

	// Port from PORT env (set in .env); default 8080.
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("Server starts on port %s\n", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		// Print the real reason (e.g. "address already in use") so failures are diagnosable.
		fmt.Println("Server failed:", err)
	}
}
