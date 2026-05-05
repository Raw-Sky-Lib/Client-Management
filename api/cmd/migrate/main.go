package main

import (
	"log"
	"os"

	"github.com/DagMT/client-portal/internal/database"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Printf("no .env file, using environment variables")
	}

	dbURL := os.Getenv("SUPABASE_DB_URL")
	if dbURL == "" {
		log.Fatal("SUPABASE_DB_URL is not set")
	}

	pool, err := database.Connect(dbURL)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	if err := database.MigratePortalDB(pool); err != nil {
		log.Fatalf("migrations failed: %v", err)
	}

	log.Println("portal migrations complete")
}
