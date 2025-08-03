package main

import (
	"database/sql"
	// "fmt"
	"log"
	// "net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

type Receipt struct {
	ID          int    `json:"id" db:"id"`
	Name        string `json:"name" db:"name"`
	Ingredients string `json:"ingredients" db:"ingredients"`
	Algorithm   string `json:"algorithm" db:"algorithm"`
	Author      string `json:"author" db:"author"`
}

var db *sql.DB

func initDB() {
	var err error
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:posrtgrespassword@localhost/receiptkins?sslmode=disable"
	}

	db, err = sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	if err = db.Ping(); err != nil {
		log.Fatal("Failed to ping database:", err)
	}

	log.Println("Database connected successfully")
}

func main() {
	initDB()
	defer db.Close()

	r := gin.Default()
	r.LoadHTMLGlob("templates/*")
	r.Static("/static", "./static")

	// Routes
	r.GET("/", homePage)
	r.GET("/search", searchReceipts)
	r.GET("/recipe/:id", recipePage)
	r.GET("/create", createPage)
	r.POST("/api/receipts", createReceipt)

	log.Println("Server starting on :8080")
	r.Run(":8080")
}

func homePage(c *gin.Context) {
	receipts, err := getAllReceipts()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.HTML(200, "index.html", gin.H{
		"receipts": receipts,
	})
}

func searchReceipts(c *gin.Context) {
	query := c.Query("q")
	receipts, err := searchReceiptsByName(query)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.HTML(200, "recipe-grid.html", gin.H{
		"receipts": receipts,
	})
}

func recipePage(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid ID"})
		return
	}

	receipt, err := getReceiptByID(id)
	if err != nil {
		c.JSON(404, gin.H{"error": "Recipe not found"})
		return
	}

	// Parse ingredients
	ingredients := parseIngredients(receipt.Ingredients)

	c.HTML(200, "recipe.html", gin.H{
		"receipt":     receipt,
		"ingredients": ingredients,
	})
}

func createPage(c *gin.Context) {
	c.HTML(200, "create.html", nil)
}

func createReceipt(c *gin.Context) {
	var receipt Receipt
	if err := c.ShouldBindJSON(&receipt); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	err := insertReceipt(&receipt)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(201, receipt)
}

// Database functions
func getAllReceipts() ([]Receipt, error) {
	rows, err := db.Query("SELECT id, name, ingredients, algorithm, author FROM receipts ORDER BY id DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var receipts []Receipt
	for rows.Next() {
		var r Receipt
		err := rows.Scan(&r.ID, &r.Name, &r.Ingredients, &r.Algorithm, &r.Author)
		if err != nil {
			return nil, err
		}
		receipts = append(receipts, r)
	}

	return receipts, nil
}

func searchReceiptsByName(query string) ([]Receipt, error) {
	rows, err := db.Query("SELECT id, name, ingredients, algorithm, author FROM receipts WHERE name ILIKE $1 ORDER BY id DESC", "%"+query+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var receipts []Receipt
	for rows.Next() {
		var r Receipt
		err := rows.Scan(&r.ID, &r.Name, &r.Ingredients, &r.Algorithm, &r.Author)
		if err != nil {
			return nil, err
		}
		receipts = append(receipts, r)
	}

	return receipts, nil
}

func getReceiptByID(id int) (*Receipt, error) {
	var r Receipt
	err := db.QueryRow("SELECT id, name, ingredients, algorithm, author FROM receipts WHERE id = $1", id).
		Scan(&r.ID, &r.Name, &r.Ingredients, &r.Algorithm, &r.Author)
	if err != nil {
		return nil, err
	}

	return &r, nil
}

func insertReceipt(receipt *Receipt) error {
	err := db.QueryRow("INSERT INTO receipts (name, ingredients, algorithm, author) VALUES ($1, $2, $3, $4) RETURNING id",
		receipt.Name, receipt.Ingredients, receipt.Algorithm, receipt.Author).Scan(&receipt.ID)
	return err
}

func parseIngredients(ingredients string) []map[string]string {
	var result []map[string]string
	items := strings.Split(ingredients, "|")

	for _, item := range items {
		if strings.TrimSpace(item) == "" {
			continue
		}
		parts := strings.Split(item, ":")
		if len(parts) == 2 {
			result = append(result, map[string]string{
				"name":     strings.TrimSpace(parts[0]),
				"quantity": strings.TrimSpace(parts[1]),
			})
		}
	}

	return result
}
