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

// Task represents the task model
type Task struct {
	ID          int        `json:"id"`
	UserID      int        `json:"user_id"`
	UserEmail   string     `json:"user_email,omitempty"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      string     `json:"status"`   // "pending", "in_progress", "completed"
	Priority    string     `json:"priority"` // "low", "medium", "high"
	DueDate     *time.Time `json:"due_date"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// CreateTaskRequest holds request body for creating a task
type CreateTaskRequest struct {
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      string     `json:"status"`   // default "pending"
	Priority    string     `json:"priority"` // default "medium"
	DueDate     *time.Time `json:"due_date"`
}

// UpdateTaskRequest holds fields that can be updated
type UpdateTaskRequest struct {
	Title       *string    `json:"title"`
	Description *string    `json:"description"`
	Status      *string    `json:"status"`
	Priority    *string    `json:"priority"`
	DueDate     *time.Time `json:"due_date"`
}

// PaginatedTasksResponse is the payload returned for listing tasks
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

// CreateTaskHandler handles POST /tasks
func CreateTaskHandler(w http.ResponseWriter, r *http.Request) {
	userID, _ := GetUserID(r.Context())

	var req CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid JSON request body")
		return
	}

	// Validation
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

	// Write Activity Log
	LogActivity(ctx, task.ID, userID, "created", map[string]interface{}{
		"title":       task.Title,
		"priority":    task.Priority,
		"status":      task.Status,
		"due_date":    task.DueDate,
		"description": task.Description,
	})

	respondWithJSON(w, http.StatusCreated, task)
}

// GetTasksHandler handles GET /tasks (filters, search, sorting, pagination)
func GetTasksHandler(w http.ResponseWriter, r *http.Request) {
	userID, _ := GetUserID(r.Context())
	role, _ := GetUserRole(r.Context())

	// Query params
	statusFilter := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("status")))
	searchQuery := strings.TrimSpace(r.URL.Query().Get("search"))
	sortBy := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("sort_by")))
	order := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("order")))

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 100 {
		limit = 10
	}
	offset := (page - 1) * limit

	ctx := r.Context()

	// Dynamic Query Assembly
	querySelect := `
		SELECT t.id, t.user_id, u.email, t.title, t.description, t.status, t.priority, t.due_date, t.created_at, t.updated_at
		FROM tasks t
		JOIN users u ON t.user_id = u.id
	`
	queryCount := `
		SELECT COUNT(*)
		FROM tasks t
		JOIN users u ON t.user_id = u.id
	`

	whereClauses := []string{}
	args := []interface{}{}
	argCount := 1

	// Access control (standard user: only own tasks, admin: all tasks)
	if role != "admin" {
		whereClauses = append(whereClauses, fmt.Sprintf("t.user_id = $%d", argCount))
		args = append(args, userID)
		argCount++
	}

	// Filters
	if statusFilter != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("t.status = $%d", argCount))
		args = append(args, statusFilter)
		argCount++
	}

	// Search
	if searchQuery != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("t.title ILIKE $%d", argCount))
		args = append(args, "%"+searchQuery+"%")
		argCount++
	}

	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = " WHERE " + strings.Join(whereClauses, " AND ")
		querySelect += whereSQL
		queryCount += whereSQL
	}

	// Sorting
	if order != "asc" && order != "desc" {
		order = "desc" // Default to descending
	}

	sortSQL := ""
	switch sortBy {
	case "due_date":
		sortSQL = "t.due_date"
		// If ordering not specified, due dates are usually asc (earliest first), but handle NULL due dates last
		sortSQL += " ASC NULLS LAST"
		if r.URL.Query().Get("order") == "desc" {
			sortSQL = "t.due_date DESC NULLS LAST"
		}
	case "priority":
		sortSQL = "CASE t.priority WHEN 'low' THEN 1 WHEN 'medium' THEN 2 WHEN 'high' THEN 3 END " + order
	case "created_at":
		sortSQL = "t.created_at " + order
	default:
		sortSQL = "t.created_at " + order
	}

	querySelect += " ORDER BY " + sortSQL

	// Count total items
	var totalItems int
	err := DB.QueryRow(ctx, queryCount, args...).Scan(&totalItems)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to count tasks: "+err.Error())
		return
	}

	// Append pagination limit & offset
	querySelect += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argCount, argCount+1)
	args = append(args, limit, offset)

	// Fetch data
	rows, err := DB.Query(ctx, querySelect, args...)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to retrieve tasks: "+err.Error())
		return
	}
	defer rows.Close()

	tasks := []Task{}
	for rows.Next() {
		var t Task
		err := rows.Scan(&t.ID, &t.UserID, &t.UserEmail, &t.Title, &t.Description, &t.Status, &t.Priority, &t.DueDate, &t.CreatedAt, &t.UpdatedAt)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to parse task: "+err.Error())
			return
		}
		tasks = append(tasks, t)
	}

	totalPages := totalItems / limit
	if totalItems%limit > 0 {
		totalPages++
	}

	respondWithJSON(w, http.StatusOK, PaginatedTasksResponse{
		Data: tasks,
		Pagination: Pagination{
			TotalItems:  totalItems,
			TotalPages:  totalPages,
			CurrentPage: page,
			Limit:       limit,
		},
	})
}

// GetTaskHandler handles GET /tasks/:id
func GetTaskHandler(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	taskID, err := strconv.Atoi(idStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid task ID")
		return
	}

	userID, _ := GetUserID(r.Context())
	role, _ := GetUserRole(r.Context())

	ctx := r.Context()
	var t Task

	query := `
		SELECT t.id, t.user_id, u.email, t.title, t.description, t.status, t.priority, t.due_date, t.created_at, t.updated_at
		FROM tasks t
		JOIN users u ON t.user_id = u.id
		WHERE t.id = $1
	`
	err = DB.QueryRow(ctx, query, taskID).
		Scan(&t.ID, &t.UserID, &t.UserEmail, &t.Title, &t.Description, &t.Status, &t.Priority, &t.DueDate, &t.CreatedAt, &t.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			respondWithError(w, http.StatusNotFound, "Task not found")
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Error reading task: "+err.Error())
		return
	}

	// Verify permissions
	if role != "admin" && t.UserID != userID {
		respondWithError(w, http.StatusForbidden, "You do not have permission to view this task")
		return
	}

	respondWithJSON(w, http.StatusOK, t)
}

// UpdateTaskHandler handles PATCH /tasks/:id
func UpdateTaskHandler(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	taskID, err := strconv.Atoi(idStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid task ID")
		return
	}

	userID, _ := GetUserID(r.Context())
	role, _ := GetUserRole(r.Context())

	var req UpdateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid JSON request body")
		return
	}

	ctx := r.Context()

	// 1. Fetch current task details for change validation & logging
	var oldTask Task
	querySelect := `SELECT id, user_id, title, description, status, priority, due_date FROM tasks WHERE id = $1`
	err = DB.QueryRow(ctx, querySelect, taskID).
		Scan(&oldTask.ID, &oldTask.UserID, &oldTask.Title, &oldTask.Description, &oldTask.Status, &oldTask.Priority, &oldTask.DueDate)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			respondWithError(w, http.StatusNotFound, "Task not found")
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Database error: "+err.Error())
		return
	}

	// 2. Access control check
	if role != "admin" && oldTask.UserID != userID {
		respondWithError(w, http.StatusForbidden, "You do not have permission to modify this task")
		return
	}

	// 3. Assemble PATCH update values dynamically
	updateFields := []string{}
	args := []interface{}{}
	argCount := 1

	updatedTask := oldTask // copy details to compare logs later

	if req.Title != nil {
		titleVal := strings.TrimSpace(*req.Title)
		if titleVal == "" {
			respondWithError(w, http.StatusBadRequest, "Task title cannot be updated to empty")
			return
		}
		updateFields = append(updateFields, fmt.Sprintf("title = $%d", argCount))
		args = append(args, titleVal)
		argCount++
		updatedTask.Title = titleVal
	}

	if req.Description != nil {
		descVal := strings.TrimSpace(*req.Description)
		updateFields = append(updateFields, fmt.Sprintf("description = $%d", argCount))
		args = append(args, descVal)
		argCount++
		updatedTask.Description = descVal
	}

	if req.Status != nil {
		statusVal := strings.TrimSpace(strings.ToLower(*req.Status))
		if statusVal != "pending" && statusVal != "in_progress" && statusVal != "completed" {
			respondWithError(w, http.StatusBadRequest, "Status must be 'pending', 'in_progress', or 'completed'")
			return
		}
		updateFields = append(updateFields, fmt.Sprintf("status = $%d", argCount))
		args = append(args, statusVal)
		argCount++
		updatedTask.Status = statusVal
	}

	if req.Priority != nil {
		priorityVal := strings.TrimSpace(strings.ToLower(*req.Priority))
		if priorityVal != "low" && priorityVal != "medium" && priorityVal != "high" {
			respondWithError(w, http.StatusBadRequest, "Priority must be 'low', 'medium', or 'high'")
			return
		}
		updateFields = append(updateFields, fmt.Sprintf("priority = $%d", argCount))
		args = append(args, priorityVal)
		argCount++
		updatedTask.Priority = priorityVal
	}

	// Note: DueDate can be updated to nil, which is valid
	if req.DueDate != nil {
		updateFields = append(updateFields, fmt.Sprintf("due_date = $%d", argCount))
		args = append(args, req.DueDate)
		argCount++
		updatedTask.DueDate = req.DueDate
	}

	if len(updateFields) == 0 {
		// Nothing to update, just return current task
		respondWithJSON(w, http.StatusOK, oldTask)
		return
	}

	// Always update updated_at timestamp
	updateFields = append(updateFields, fmt.Sprintf("updated_at = NOW()"))

	queryUpdate := fmt.Sprintf("UPDATE tasks SET %s WHERE id = $%d", strings.Join(updateFields, ", "), argCount)
	args = append(args, taskID)

	_, err = DB.Exec(ctx, queryUpdate, args...)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to update task: "+err.Error())
		return
	}

	// Write Activity Log
	diff := buildTaskDiffDetails(oldTask, updatedTask)
	if len(diff) > 0 {
		LogActivity(ctx, taskID, userID, "updated", diff)
	}

	// Fetch fully updated model to return
	var finalTask Task
	queryFinal := `
		SELECT t.id, t.user_id, u.email, t.title, t.description, t.status, t.priority, t.due_date, t.created_at, t.updated_at
		FROM tasks t
		JOIN users u ON t.user_id = u.id
		WHERE t.id = $1
	`
	err = DB.QueryRow(ctx, queryFinal, taskID).
		Scan(&finalTask.ID, &finalTask.UserID, &finalTask.UserEmail, &finalTask.Title, &finalTask.Description, &finalTask.Status, &finalTask.Priority, &finalTask.DueDate, &finalTask.CreatedAt, &finalTask.UpdatedAt)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to read updated task state")
		return
	}

	respondWithJSON(w, http.StatusOK, finalTask)
}

// DeleteTaskHandler handles DELETE /tasks/:id
func DeleteTaskHandler(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	taskID, err := strconv.Atoi(idStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid task ID")
		return
	}

	userID, _ := GetUserID(r.Context())
	role, _ := GetUserRole(r.Context())

	ctx := r.Context()

	// 1. Fetch user_id for check
	var taskOwnerID int
	query := `SELECT user_id FROM tasks WHERE id = $1`
	err = DB.QueryRow(ctx, query, taskID).Scan(&taskOwnerID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			respondWithError(w, http.StatusNotFound, "Task not found")
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Database error: "+err.Error())
		return
	}

	// 2. Verify permissions
	if role != "admin" && taskOwnerID != userID {
		respondWithError(w, http.StatusForbidden, "You do not have permission to delete this task")
		return
	}

	// Delete from DB (associated activity logs cascade-delete automatically)
	_, err = DB.Exec(ctx, "DELETE FROM tasks WHERE id = $1", taskID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to delete task: "+err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
