package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"./models"
	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/joho/godotenv"
)

func main() {

	r := mux.NewRouter()

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	dbURI := fmt.Sprintf("host=%s port=%s user=%s dbname=%s password=%s sslmode=disable",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_PASSWORD"))

	db, err := gorm.Open("postgres", dbURI)
	if err != nil {
		panic("failed to connect db")
	}
	db.AutoMigrate(&models.User{})

	defer db.Close()

	r.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
		user := models.User{}

		decoder := json.NewDecoder(r.Body)
		err := decoder.Decode(&user)
		if err != nil {
			errorResponse(w, http.StatusBadRequest, err.Error())
		}

		if err := db.Create(&user).Error; err != nil {
			errorResponse(w, http.StatusInternalServerError, err.Error())
		}

		toResponse(w, 200, user)
	}).Methods("POST")

	_ = http.ListenAndServe(":8080", r)

}

func toResponse(w http.ResponseWriter, responseCode int, responseJSON interface{}) {
	response, err := json.Marshal(responseJSON)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	w.WriteHeader(responseCode)
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(response)
}

func errorResponse(w http.ResponseWriter, responseCode int, error string) {
	toResponse(w, responseCode, map[string]string{"error": error})
}
