package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"./models"
	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
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

		user.Password, _ = GenerateHash(user.Password)
		token, _ := GenerateToken(user.Name)

		if err := db.Create(&user).Error; err != nil {
			errorResponse(w, http.StatusInternalServerError, err.Error())
		}

		cookie := http.Cookie{Name: "token", Value: token}
		http.SetCookie(w, &cookie)
		toResponse(w, 200, map[string]string{"result": "success"})
	}).Methods("POST")

	r.HandleFunc("/session", func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("token")
		if err != nil {
			errorResponse(w, 401, "Unauthorized")
		}

		token, err := jwt.Parse(cookie.Value, func(token *jwt.Token) (interface{}, error) {
			if token != nil {
			}
			return []byte(os.Getenv("SECRET_KEY")), nil
		})

		if token != nil {
			if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
				toResponse(w, 200, claims)
			}
		} else {
			errorResponse(w, 401, "Unauthorized")
		}

	}).Methods("GET")

	r.HandleFunc("/session", func(w http.ResponseWriter, r *http.Request) {
		user := models.User{}

		decoder := json.NewDecoder(r.Body)
		err := decoder.Decode(&user)
		if err != nil {
			errorResponse(w, http.StatusBadRequest, err.Error())
		}

		userFromDb := GetUser(db, user.Name, w)
		if userFromDb == nil {
			return
		}

		match := CheckPasswordHash(user.Password, userFromDb.Password)

		if match == true {
			token, _ := GenerateToken(user.Name)
			cookie := http.Cookie{Name: "token", Value: token}
			http.SetCookie(w, &cookie)
			toResponse(w, 200, map[string]string{"result": "success"})
		} else {
			errorResponse(w, 200, "Incorrect username or password")
		}

	}).Methods("POST")

	r.HandleFunc("/session", func(w http.ResponseWriter, r *http.Request) {
		cookie := http.Cookie{Name: "token", MaxAge: -1}
		http.SetCookie(w, &cookie)
		toResponse(w, 200, map[string]string{"result": "success"})
	}).Methods("DELETE")

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

func GenerateHash(password string) (string, error) {
	bytePassword := []byte(password)

	hash, err := bcrypt.GenerateFromPassword(bytePassword, bcrypt.MinCost)

	return string(hash), err
}

func GenerateToken(username string) (string, error) {
	secretKey := []byte(os.Getenv("SECRET_KEY"))

	claims := &jwt.StandardClaims{
		ExpiresAt: time.Now().Unix() + 86400,
		IssuedAt:  time.Now().Unix(),
		Subject:   username,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	ss, err := token.SignedString(secretKey)

	return ss, err
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func GetUser(db *gorm.DB, username string, w http.ResponseWriter) *models.User {
	user := models.User{}
	if err := db.Find(&user, models.User{Name: username}).Error; err != nil {
		errorResponse(w, 200, "Incorrect username or password")
		return nil
	}
	return &user
}
