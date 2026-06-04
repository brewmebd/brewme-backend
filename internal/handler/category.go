package handler

import (
	"brewme/internal/database"
	"brewme/internal/model"
	"encoding/json"
	"net/http"
)

func GetAllCategory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req []model.GetAllCategory
	query := `SELECT id, name, slug, created_at FROM categories`
	rows, err := database.DB.Query(query)
	if err != nil {
		http.Error(w, "Error to query on database", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var reqc model.GetAllCategory
		err = rows.Scan(&reqc.CategoryID, &reqc.CategoryName, &reqc.CategorySlug, &reqc.CreatedAt)
		if err != nil {
			http.Error(w, "Invalid Scan", http.StatusInternalServerError)
			return
		}
		req = append(req, reqc)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":   true,
		"category": req,
	})

}
