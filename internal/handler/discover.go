package handler

import (
	"brewme/internal/database"
	"brewme/internal/model"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

func GetAllCreator(w http.ResponseWriter, r *http.Request) {
	var creator []model.GetAllCreator

	query := `SELECT 
    u.id AS creator_id,
    u.username AS creator_username,
    u.avatar_url AS creator_profile_picture,
    u.full_name AS creator_name,
    c.name AS creator_category,
    u.bio AS creator_description,
    COALESCE(SUM(d.cups), 0) AS total_supporters_cup
FROM 
    users u
LEFT JOIN 
    categories c ON u.category_id = c.id
LEFT JOIN 
    donations d ON u.id = d.user_id AND d.status = 'succeeded'
GROUP BY 
    u.id, 
    u.username, 
    u.avatar_url, 
    u.full_name, 
    c.name, 
    u.bio
ORDER BY 
    u.id ASC;`

	rows, err := database.DB.Query(query)
	if err != nil {
		http.Error(w, "Failed to execute database query", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var c model.GetAllCreator
		err := rows.Scan(
			&c.CreatorID,
			&c.CreatorUsername,
			&c.CreatorProfilePicture,
			&c.CreatorName,
			&c.CreatorCategory,
			&c.CreatorDescription,
			&c.TotalSupportersCup,
		)
		if err != nil {
			http.Error(w, "Error parsing database data", http.StatusInternalServerError)
			fmt.Println(err)
			return
		}

		creator = append(creator, c)
	}

	if err = rows.Err(); err != nil {
		http.Error(w, "Error reading database rows", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(creator); err != nil {
		log.Printf("Error encoding feed response: %v", err)
	}
}
