package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

type Task struct {
	ID          int        `json:"id"`
	UserID      int        `json:"user_id"`
	UserEmail   string     `json:"user_email,omitempty"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      string     `json:"status"`
	Priority    string     `json:"priority"`
	DueDate     *time.Time `json:"due_date"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type CreateTaskRequest struct {
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      string     `json:"status"`
	Priority    string     `json:"priority"`
	DueDate     *time.Time `json:"due_date"`
}

type UpdateTaskRequest struct {
	Title       *string    `json:"title"`
	Description *string    `json:"description"`
	Status      *string    `json:"status"`
	Priority    *string    `json:"priority"`
	DueDate     *time.Time `json:"due_date"`
}

type PaginatedTasksResponse struct {
	Data       []Task     `json:"data"`
	Pagination Pagination `json:"pagination"`
}

type Pagination struct {
	TotalItems  int `json:"total_items"`
	TotalPages  int `json:"total_pages"`
	CurrentPage int `json:"current_page"`
	Limit       int `json:"limit"`
}

func CreateTaskHandler(w http.ResponseWriter, r *http.Request) {
	userID, _ := GetUserID(r.Context())

	var req CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid JSON request body")
		return
	}

	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		respondWithError(w, http.StatusBadRequest, "Task title is required")
		return
	}

	req.Status = strings.TrimSpace(strings.ToLower(req.Status))
	if req.Status == "" {
		req.Status = "pending"
	}
	if req.Status != "pending" && req.Status != "in_progress" && req.Status != "completed" {
		respondWithError(w, http.StatusBadRequest, "Status must be 'pending', 'in_progress', or 'completed'")
		return
	}

	req.Priority = strings.TrimSpace(strings.ToLower(req.Priority))
	if req.Priority == "" {
		req.Priority = "medium"
	}
	if req.Priority != "low" && req.Priority != "medium" && req.Priority != "high" {
		respondWithError(w, http.StatusBadRequest, "Priority must be 'low', 'medium', or 'high'")
		return
	}

	ctx := r.Context()
	var task Task

	query := `
		INSERT INTO tasks (user_id, title, description, status, priority, due_date)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, user_id, title, description, status, priority, due_date, created_at, updated_at
	`
	err := DB.QueryRow(ctx, query, userID, req.Title, req.Description, req.Status, req.Priority, req.DueDate).
		Scan(&task.ID, &task.UserID, &task.Title, &task.Description, &task.Status, &task.Priority, &task.DueDate, &task.CreatedAt, &task.UpdatedAt)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to create task: "+err.Error())
		return
	}

	LogActivity(ctx, task.ID, userID, "created", map[string]interface{}{
		"title":       task.Title,
		"priority":    task.Priority,
		"status":      task.Status,
		"due_date":    task.DueDate,
		"description": task.Description,
	})

	respondWithJSON(w, http.StatusCreated, task)
}
