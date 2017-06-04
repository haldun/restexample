package main

import (
	"net/http"
	"strconv"
	"time"

	"database/sql"

	"github.com/dimfeld/httptreemux"
	_ "github.com/lib/pq"

	"encoding/json"

	"github.com/Sirupsen/logrus"
)

type Product struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Price       float64   `json:"price"`
	CreatedAt   time.Time `json:"created_at"`
}

type ProductStore interface {
	AllProducts() ([]*Product, error)
	FindProduct(ID int) (*Product, error)
}

type DataStore interface {
	ProductStore
}

type DB struct {
	*sql.DB
}

func NewDB(dataSourceName string) (*DB, error) {
	db, err := sql.Open("postgres", dataSourceName)
	if err != nil {
		return nil, err
	}
	if err = db.Ping(); err != nil {
		return nil, err
	}
	return &DB{db}, nil
}

func (db *DB) AllProducts() ([]*Product, error) {
	rows, err := db.Query(`select id, title, description, price, created_at from products`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	products := []*Product{}
	for rows.Next() {
		product := new(Product)
		err := rows.Scan(&product.ID, &product.Title, &product.Description, &product.Price, &product.CreatedAt)
		if err != nil {
			return nil, err
		}
		products = append(products, product)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return products, nil
}

func (db *DB) FindProduct(ID int) (*Product, error) {
	query := `
		select id, title, description, price, created_at
		from products
		where id = $1
	`
	row := db.QueryRow(query, ID)
	product := new(Product)
	if err := row.Scan(&product.ID, &product.Title, &product.Description, &product.Price, &product.CreatedAt); err != nil {
		return nil, err
	}
	return product, nil
}

type Server struct {
	db DataStore
}

func (s *Server) Serve() error {
	r := httptreemux.NewContextMux()
	g := r.NewGroup("/products")
	g.GET("/", s.listProducts)
	g.GET("/:id", s.showProduct)
	g.POST("/", createProduct)
	g.PATCH("/", updateProduct)
	g.DELETE("/:id", deleteProduct)
	return http.ListenAndServe(":8080", r)
}

func (s *Server) listProducts(w http.ResponseWriter, r *http.Request) {
	products, err := s.db.AllProducts()
	if err != nil {
		logrus.Errorf("error fetching products: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	if err = json.NewEncoder(w).Encode(products); err != nil {
		logrus.Errorf("error encoding products: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}

func (s *Server) showProduct(w http.ResponseWriter, r *http.Request) {
	params := httptreemux.ContextParams(r.Context())
	id, err := strconv.Atoi(params["id"])
	if err != nil {
		logrus.Errorf("could not parse ID: %v", err)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	product, err := s.db.FindProduct(id)
	if err != nil {
		switch err {
		case sql.ErrNoRows:
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		default:
			logrus.Errorf("error fetching product: %v", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}
	if err = json.NewEncoder(w).Encode(product); err != nil {
		logrus.Errorf("error encoding product: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}

func createProduct(w http.ResponseWriter, r *http.Request) {

}

func updateProduct(w http.ResponseWriter, r *http.Request) {

}

func deleteProduct(w http.ResponseWriter, r *http.Request) {

}

func main() {
	db, err := NewDB("user=hb password='' dbname=demo sslmode=disable")
	if err != nil {
		logrus.Fatalf("could not connect to database: %v", err)
	}

	server := &Server{db}
	logrus.Fatal(server.Serve())
}
