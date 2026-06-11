package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var DB *pgxpool.Pool

func InitDB() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbHost := getEnv("DB_HOST", "localhost")
		dbPort := getEnv("DB_PORT", "5432")
		dbUser := getEnv("DB_USER", "postgres")
		dbPass := getEnv("DB_PASSWORD", "postgres")
		dbName := getEnv("DB_NAME", "taskmanager")
		dbSSL := getEnv("DB_SSLMODE", "disable")
		dbURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", dbUser, dbPass, dbHost, dbPort, dbName, dbSSL)
	}

	config, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		log.Fatalf("Unable to parse database configuration: %v", err)
	}

	// Configure connection pooling details
	config.MaxConns = 10
	config.MinConns = 2
	config.MaxConnIdleTime = 30 * time.Minute

	// Connection retry logic (essential for Docker Compose startup coordination)
	var pool *pgxpool.Pool
	for i := 0; i < 15; i++ {
		pool, err = pgxpool.NewWithConfig(context.Background(), config)
		if err == nil {
			err = pool.Ping(context.Background())
			if err == nil {
				break
			}
		}
		log.Printf("Waiting for database connection (attempt %d/15)... error: %v", i+1, err)
		time.Sleep(2 * time.Second)
	}

	if err != nil {
		log.Fatalf("Unable to connect to database after retries: %v", err)
	}

	DB = pool
	log.Println("Connected to PostgreSQL successfully")

	if err := runMigrations(); err != nil {
		log.Fatalf("Database migration failed: %v", err)
	}
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return fallback
}

func runMigrations() error {
	ctx := context.Background()

	// 1. Create users table (id, email, password_hash, role, created_at)
	usersSchema := `
	CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		email VARCHAR(255) UNIQUE NOT NULL,
		password_hash VARCHAR(255) NOT NULL,
		role VARCHAR(50) NOT NULL DEFAULT 'user',
		created_at TIMESTAMP NOT NULL DEFAULT NOW()
	);`
	_, err := DB.Exec(ctx, usersSchema)
	if err != nil {
		return fmt.Errorf("failed to create users table: %w", err)
	}

	// 2. Create tasks table (id, user_id, title, description, status, priority, due_date, created_at, updated_at)
	tasksSchema := `
	CREATE TABLE IF NOT EXISTS tasks (
		id SERIAL PRIMARY KEY,
		user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		title VARCHAR(255) NOT NULL,
		description TEXT,
		status VARCHAR(50) NOT NULL DEFAULT 'pending',
		priority VARCHAR(50) NOT NULL DEFAULT 'medium',
		due_date TIMESTAMP,
		created_at TIMESTAMP NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMP NOT NULL DEFAULT NOW()
	);`
	_, err = DB.Exec(ctx, tasksSchema)
	if err != nil {
		return fmt.Errorf("failed to create tasks table: %w", err)
	}

	// 3. Create activity_logs table (id, task_id, user_id, action, details, created_at)
	logsSchema := `
	CREATE TABLE IF NOT EXISTS activity_logs (
		id SERIAL PRIMARY KEY,
		task_id INTEGER NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
		user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		action VARCHAR(100) NOT NULL,
		details TEXT,
		created_at TIMESTAMP NOT NULL DEFAULT NOW()
	);`
	_, err = DB.Exec(ctx, logsSchema)
	if err != nil {
		return fmt.Errorf("failed to create activity_logs table: %w", err)
	}

	log.Println("Database tables verified and running")
	return nil
}
