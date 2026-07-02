package handler

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"brewme/internal/database"
	"brewme/internal/model"
	"brewme/internal/utils"

	"github.com/go-chi/chi"
)

const (
	postImageUploadDir = "uploads/posts"
	maxPostImageBytes  = 5 << 20 // 5MB
)

// Columns selected for every dashboard post read — keep in sync with scanPost.
const postColumns = `id, title, preview, body, image_url, visibility, status,
	likes_count, comments_count, published_at, created_at`

// allowedPostImageTypes maps an accepted image content-type to its extension.
var allowedPostImageTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/gif":  ".gif",
	"image/webp": ".webp",
}

// rowScanner is satisfied by both *sql.Row and *sql.Rows.
type rowScanner interface {
	Scan(dest ...any) error
}

// scanPost reads one row (in postColumns order) into a DashboardPostItem.
func scanPost(s rowScanner) (model.DashboardPostItem, error) {
	var p model.DashboardPostItem
	var preview, body, image sql.NullString
	var publishedAt sql.NullTime

	err := s.Scan(
		&p.ID, &p.Title, &preview, &body, &image, &p.Visibility, &p.Status,
		&p.LikesCount, &p.CommentsCount, &publishedAt, &p.CreatedAt,
	)
	if err != nil {
		return p, err
	}
	p.Preview = preview.String
	p.Body = body.String
	if image.Valid && image.String != "" {
		v := image.String
		p.Image = &v
	}
	if publishedAt.Valid {
		t := publishedAt.Time
		p.PublishedAt = &t
	}
	p.MembersOnly = p.Visibility == "members"
	return p, nil
}

// savePostImage validates and stores an uploaded post image, returning the
// public URL path (served by the /uploads/* static route).
func savePostImage(file multipart.File, header *multipart.FileHeader) (string, error) {
	if header.Size > maxPostImageBytes {
		return "", fmt.Errorf("image exceeds 5MB limit")
	}

	// Sniff the real content type from the first 512 bytes.
	head := make([]byte, 512)
	n, _ := file.Read(head)
	ext, ok := allowedPostImageTypes[http.DetectContentType(head[:n])]
	if !ok {
		return "", fmt.Errorf("image must be a JPG, PNG, GIF, or WebP")
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", err
	}

	if err := os.MkdirAll(postImageUploadDir, 0o755); err != nil {
		return "", err
	}
	name, err := randomName() // shared helper in handler.go
	if err != nil {
		return "", err
	}
	fullPath := filepath.Join(postImageUploadDir, name+ext)

	dst, err := os.Create(fullPath)
	if err != nil {
		return "", err
	}
	defer dst.Close()
	if _, err := io.Copy(dst, file); err != nil {
		_ = os.Remove(fullPath)
		return "", err
	}

	return "/" + postImageUploadDir + "/" + name + ext, nil
}

// deletePostImageFile removes an uploaded image from disk given its public URL
// (e.g. "/uploads/posts/abc.png"). Best-effort: errors are logged, not fatal.
func deletePostImageFile(url string) {
	if url == "" || !strings.HasPrefix(url, "/"+postImageUploadDir+"/") {
		return
	}
	path := strings.TrimPrefix(url, "/")
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		log.Println("Failed to delete post image:", err)
	}
}

// GetDashboardPosts returns every post belonging to the authenticated creator
// (all visibilities and statuses), newest first — for the dashboard Posts page.
func GetDashboardPosts(w http.ResponseWriter, r *http.Request) {
	userID, ok := authUserID(w, r)
	if !ok {
		return
	}

	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			if n > 100 {
				n = 100
			}
			limit = n
		}
	}

	rows, err := database.DB.Query(`
		SELECT `+postColumns+`
		FROM posts
		WHERE user_id = ?
		ORDER BY COALESCE(published_at, created_at) DESC
		LIMIT ?`,
		userID, limit,
	)
	if err != nil {
		http.Error(w, "Error fetching posts", http.StatusInternalServerError)
		log.Println("Database error (GetDashboardPosts):", err)
		return
	}
	defer rows.Close()

	posts := make([]model.DashboardPostItem, 0)
	for rows.Next() {
		p, err := scanPost(rows)
		if err != nil {
			log.Println("Error scanning post row:", err)
			continue
		}
		posts = append(posts, p)
	}
	if err = rows.Err(); err != nil {
		http.Error(w, "Error reading posts", http.StatusInternalServerError)
		log.Println("Database iteration error (GetDashboardPosts):", err)
		return
	}

	writeJSON(w, http.StatusOK, posts)
}

// CreatePost publishes (or drafts) a new post for the authenticated creator.
func CreatePost(w http.ResponseWriter, r *http.Request) {
	userID, ok := authUserID(w, r)
	if !ok {
		return
	}

	title, content, preview, visibility, status, image, ok := parsePostForm(w, r)
	if !ok {
		return
	}

	// published_at is set only when publishing; drafts have a NULL publish date.
	res, err := database.DB.Exec(`
		INSERT INTO posts (user_id, title, body, preview, image_url, visibility, status, published_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, CASE WHEN ? = 'published' THEN CURRENT_TIMESTAMP ELSE NULL END)`,
		userID, title, content, preview, image, visibility, status, status,
	)
	if err != nil {
		http.Error(w, "Error creating post", http.StatusInternalServerError)
		log.Println("Database error (CreatePost):", err)
		return
	}

	newID, err := res.LastInsertId()
	if err != nil {
		http.Error(w, "Error confirming post", http.StatusInternalServerError)
		return
	}

	p, err := loadPost(newID, userID)
	if err != nil {
		http.Error(w, "Error loading created post", http.StatusInternalServerError)
		log.Println("Database error (CreatePost readback):", err)
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

// UpdatePost edits an existing post owned by the authenticated creator.
func UpdatePost(w http.ResponseWriter, r *http.Request) {
	userID, ok := authUserID(w, r)
	if !ok {
		return
	}

	postID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid post ID", http.StatusBadRequest)
		return
	}

	// Ownership check + fetch current row (for the old image path).
	current, err := loadPost(postID, userID)
	if err == sql.ErrNoRows {
		http.Error(w, "Post not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "Error loading post", http.StatusInternalServerError)
		log.Println("Database error (UpdatePost load):", err)
		return
	}

	title, content, preview, visibility, status, newImage, ok := parsePostForm(w, r)
	if !ok {
		return
	}

	// Resolve the final image: a new upload replaces it, removeImage=true clears
	// it, otherwise keep the existing one. Track the old file for cleanup.
	oldImage := ""
	if current.Image != nil {
		oldImage = *current.Image
	}
	finalImage := newImage // set when a new file was uploaded
	if newImage == nil {
		if r.FormValue("removeImage") == "true" {
			finalImage = nil
		} else {
			finalImage = interfaceOrNil(oldImage) // keep existing
		}
	}

	// Publishing a previously-drafted post stamps published_at once.
	_, err = database.DB.Exec(`
		UPDATE posts
		SET title = ?, body = ?, preview = ?, image_url = ?, visibility = ?, status = ?,
		    published_at = CASE WHEN ? = 'published' THEN COALESCE(published_at, CURRENT_TIMESTAMP) ELSE published_at END,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND user_id = ?`,
		title, content, preview, finalImage, visibility, status, status, postID, userID,
	)
	if err != nil {
		http.Error(w, "Error updating post", http.StatusInternalServerError)
		log.Println("Database error (UpdatePost):", err)
		return
	}

	// Remove the old image file if it was replaced or cleared.
	newImageStr := ""
	if s, ok := finalImage.(string); ok {
		newImageStr = s
	}
	if oldImage != "" && oldImage != newImageStr {
		deletePostImageFile(oldImage)
	}

	p, err := loadPost(postID, userID)
	if err != nil {
		http.Error(w, "Error loading updated post", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

// DeletePost removes a post (and its image file) owned by the creator.
func DeletePost(w http.ResponseWriter, r *http.Request) {
	userID, ok := authUserID(w, r)
	if !ok {
		return
	}

	postID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid post ID", http.StatusBadRequest)
		return
	}

	// Fetch the image path first (also serves as the ownership check).
	var image sql.NullString
	err = database.DB.QueryRow(`SELECT image_url FROM posts WHERE id = ? AND user_id = ?`, postID, userID).Scan(&image)
	if err == sql.ErrNoRows {
		http.Error(w, "Post not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "Error loading post", http.StatusInternalServerError)
		log.Println("Database error (DeletePost load):", err)
		return
	}

	if _, err := database.DB.Exec(`DELETE FROM posts WHERE id = ? AND user_id = ?`, postID, userID); err != nil {
		http.Error(w, "Error deleting post", http.StatusInternalServerError)
		log.Println("Database error (DeletePost):", err)
		return
	}

	if image.Valid {
		deletePostImageFile(image.String)
	}

	writeJSON(w, http.StatusOK, map[string]any{"status": true, "message": "Post deleted"})
}

// ── helpers ──────────────────────────────────────────────────────────────

// authUserID extracts and validates the creator's user id from the bearer
// token, writing a 401 and returning ok=false on failure.
func authUserID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	token, err := utils.GetTokenFromHeader(r)
	if err != nil {
		http.Error(w, "Missing token", http.StatusUnauthorized)
		return 0, false
	}
	userID, err := utils.GetUserIDFromToken(token)
	if err != nil {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return 0, false
	}
	return userID, true
}

// loadPost reads a single owned post by id.
func loadPost(id, userID int64) (model.DashboardPostItem, error) {
	row := database.DB.QueryRow(`SELECT `+postColumns+` FROM posts WHERE id = ? AND user_id = ?`, id, userID)
	return scanPost(row)
}

// parsePostForm reads and validates the shared create/update multipart form.
// On any validation failure it writes the error response and returns ok=false.
// The returned image is non-nil only when a new file was uploaded.
func parsePostForm(w http.ResponseWriter, r *http.Request) (title, content, preview, visibility, status string, image interface{}, ok bool) {
	if err := r.ParseMultipartForm(maxPostImageBytes + (1 << 20)); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	title = strings.TrimSpace(r.FormValue("title"))
	if title == "" {
		http.Error(w, "Title is required", http.StatusBadRequest)
		return
	}

	// The dashboard sends a single content field as `preview`. Keep the full
	// text in `body` and a <=500-char snippet in `preview`.
	content = strings.TrimSpace(r.FormValue("preview"))
	if content == "" {
		content = strings.TrimSpace(r.FormValue("body"))
	}
	if content == "" {
		http.Error(w, "Content is required", http.StatusBadRequest)
		return
	}
	preview = content
	if len(preview) > 500 {
		preview = preview[:497] + "..."
	}

	membersOnly := r.FormValue("membersOnly") == "true"
	visibility = "public"
	if r.FormValue("visibility") == "members" || membersOnly {
		visibility = "members"
	}

	status = "published"
	if r.FormValue("status") == "draft" {
		status = "draft"
	}

	if file, header, ferr := r.FormFile("image"); ferr == nil {
		defer file.Close()
		url, serr := savePostImage(file, header)
		if serr != nil {
			http.Error(w, serr.Error(), http.StatusBadRequest)
			return
		}
		image = url
	} else if ferr != http.ErrMissingFile {
		http.Error(w, "Invalid image upload", http.StatusBadRequest)
		return
	}

	ok = true
	return
}

// interfaceOrNil returns s as an interface{}, or nil when empty (so it maps to
// SQL NULL rather than an empty string).
func interfaceOrNil(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
