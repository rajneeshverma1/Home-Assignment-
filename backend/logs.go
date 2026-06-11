package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
)

// ActivityLog represents a task audit log entry
type ActivityLog struct {
	ID        int       `json:"id"`
	TaskID    int       `json:"task_id"`
	UserID    int       `json:"user_id"`
	UserEmail string    `json:"user_email,omitempty"`
	Action    string    `json:"action"` // "created", "updated_title", "updated_status", etc.
	Details   string    `json:"details"`
	CreatedAt time.Time `json:"created_at"`
}

// LogActivity inserts a log record. It can run inside an active transaction.
func LogActivity(ctx context.Context, taskID int, userID int, action string, details map[string]interface{}) {
	detailsJSON, err := json.Marshal(details)
	if err != nil {
		log.Printf("Failed to marshal activity details: %v", err)
		return
	}

	query := `
		INSERT INTO activity_logs (task_id, user_id, action, details)
		VALUES ($1, $2, $3, $4)
	`
	_, err = DB.Exec(ctx, query, taskID, userID, action, string(detailsJSON))
	if err != nil {
		log.Printf("Failed to write activity log to DB: %v", err)
	}
}

// TaskLogsHandler retrieves activity logs for a specific task
func TaskLogsHandler(w http.ResponseWriter, r *http.Request) {
	taskIDStr := chi.URLParam(r, "id")
	taskID, err := strconv.Atoi(taskIDStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid task ID")
		return
	}

	userID, _ := GetUserID(r.Context())
	role, _ := GetUserRole(r.Context())

	ctx := r.Context()

	// 1. Verify task exists and user has access
	var taskOwnerID int
	err = DB.QueryRow(ctx, "SELECT user_id FROM tasks WHERE id = $1", taskID).Scan(&taskOwnerID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Task not found")
		return
	}

	if role != "admin" && taskOwnerID != userID {
		respondWithError(w, http.StatusForbidden, "You do not have permission to view logs for this task")
		return
	}

	// 2. Query logs
	query := `
		SELECT l.id, l.task_id, l.user_id, u.email, l.action, l.details, l.created_at
		FROM activity_logs l
		JOIN users u ON l.user_id = u.id
		WHERE l.task_id = $1
		ORDER BY l.created_at DESC
	`
	rows, err := DB.Query(ctx, query, taskID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to retrieve activity logs: "+err.Error())
		return
	}
	defer rows.Close()

	logs := []ActivityLog{}
	for rows.Next() {
		var l ActivityLog
		err := rows.Scan(&l.ID, &l.TaskID, &l.UserID, &l.UserEmail, &l.Action, &l.Details, &l.CreatedAt)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to scan activity log: "+err.Error())
			return
		}
		logs = append(logs, l)
	}

	respondWithJSON(w, http.StatusOK, logs)
}

// Helper function to formats diffs as human-readable activity details
func buildTaskDiffDetails(oldTask, newTask Task) map[string]interface{} {
	changes := make(map[string]interface{})
	if oldTask.Title != newTask.Title {
		changes["title"] = map[string]string{"from": oldTask.Title, "to": newTask.Title}
	}
	if oldTask.Description != newTask.Description {
		changes["description"] = map[string]string{"from": oldTask.Description, "to": newTask.Description}
	}
	if oldTask.Status != newTask.Status {
		changes["status"] = map[string]string{"from": oldTask.Status, "to": newTask.Status}
	}
	if oldTask.Priority != newTask.Priority {
		changes["priority"] = map[string]string{"from": oldTask.Priority, "to": newTask.Priority}
	}
	// Handle due date changes safely
	oldDue := ""
	if oldTask.DueDate != nil {
		oldDue = oldTask.DueDate.Format(time.RFC3339)
	}
	newDue := ""
	if newTask.DueDate != nil {
		newDue = newTask.DueDate.Format(time.RFC3339)
	}
	if oldDue != newDue {
		changes["due_date"] = map[string]string{"from": oldDue, "to": newDue}
	}
	return changes
}
