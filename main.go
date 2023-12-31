package main

import (
	"database/sql"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
)

func env() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
}

var DB *sql.DB

func openDB() error {
	db, err := sql.Open("sqlite3", os.Getenv("PRISMA_DB"))
	if err != nil {
		return err
	}
	DB = db
	return nil
}

func closeDB() error {
	return DB.Close()
}

const path = "./src"

func main() {
	env()
	openDB()
	defer closeDB()

	mux := http.NewServeMux()
	mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir(path+"/assets/"))))

	mux.HandleFunc("/", Home)
	mux.HandleFunc("/blog/", BlogId)

	mux.HandleFunc("/getBlogs/", GetBlogs)
	mux.HandleFunc("/getBlog/", GetBlog)
	mux.HandleFunc("/createBlog/", createBlog)
	mux.HandleFunc("/deleteBlog/", deleteBlog)

	err := http.ListenAndServe(":8000", addCORS(mux))
	if err != nil {
		log.Fatal(err)
	}
}
func addCORS(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		h.ServeHTTP(w, r)
	})
}

func Home(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	tmpl := template.Must(template.ParseFiles(path + "/index.html"))

	if err := tmpl.Execute(w, nil); err != nil {
		http.Error(w, "Error rendering template", http.StatusInternalServerError)
		return
	}
}

func BlogId(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	tmpl := template.Must(template.ParseFiles(path + "/getBlog.html"))

	if err := tmpl.Execute(w, nil); err != nil {
		http.Error(w, "Error rendering template", http.StatusInternalServerError)
		return
	}
}

type Blog struct {
	Id      int    `json:"id"`
	Title   string `json:"title"`
	Article string `json:"article"`
}

func GetBlogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	rows, err := DB.Query(`SELECT * FROM "Blog"`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	blogs := []Blog{}
	for rows.Next() {
		var blog Blog
		err := rows.Scan(&blog.Id, &blog.Title, &blog.Article)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		blogs = append(blogs, blog)
	}

	jsonData, err := json.Marshal(blogs)
	if err != nil {
		http.Error(w, "Unable to marshal JSON", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(jsonData)
}

func GetBlog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := r.URL.Path[len("/getBlog/"):]
	row := DB.QueryRow(`SELECT * FROM "Blog" WHERE id=$1`, id)

	var blog Blog
	err := row.Scan(&blog.Id, &blog.Title, &blog.Article)
	if err != nil {
		http.Error(w, "Error fetching blog", http.StatusInternalServerError)
		return
	}

	jsonData, err := json.Marshal(blog)
	if err != nil {
		http.Error(w, "Unable to marshal JSON", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(jsonData)
}

func createBlog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var blog Blog
	json.NewDecoder(r.Body).Decode(&blog)

	err := DB.QueryRow(`INSERT INTO "Blog" (title, article) VALUES ($1, $2) RETURNING id`, blog.Title, blog.Article).Scan(&blog.Id)
	if err != nil {
		log.Fatal(err)
	}

	json.NewEncoder(w).Encode(blog)
}

func deleteBlog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := r.URL.Path[len("/deleteBlog/"):]

	_, err := DB.Exec(`DELETE FROM "Blog" WHERE id = $1`, id)
	if err != nil {
		http.Error(w, "Error deleting todo", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
