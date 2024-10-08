package main

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type Thing struct {
	Description string `json:"shortDescription"`
	Cost        string `json:"price"`
}

type ShoppingExperience struct {
	Store      string  `json:"retailer"`
	DateOfBuy  string  `json:"purchaseDate"`
	TimeOfBuy  string  `json:"purchaseTime"`
	Stuff      []Thing `json:"items"`
	GrandTotal string  `json:"total"`
}

type MemoryBank struct {
	ShoppingData ShoppingExperience
	Score        int
}

var brain = make(map[string]MemoryBank)
var sharedBrain = &sync.Mutex{}

func funkyAlphanumericCheck(r rune) bool {
	return (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
}

func oddTimeCalculator(experience ShoppingExperience) int {
	totalScore := 0

	for _, r := range experience.Store {
		if funkyAlphanumericCheck(r) {
			totalScore++
		}
	}

	total, err := strconv.ParseFloat(experience.GrandTotal, 64)
	if err != nil {
		return totalScore
	}

	if total == float64(int(total)) {
		totalScore += 50
	}

	if math.Mod(total, 0.25) == 0 {
		totalScore += 25
	}

	totalScore += (len(experience.Stuff) / 2) * 5

	for _, thing := range experience.Stuff {
		desc := strings.TrimSpace(thing.Description)
		if len(desc)%3 == 0 {
			price, err := strconv.ParseFloat(thing.Cost, 64)
			if err != nil {
				continue
			}
			totalScore += int(math.Ceil(price * 0.2))
		}
	}

	date, err := time.Parse("2006-01-02", experience.DateOfBuy)
	if err == nil && date.Day()%2 == 1 {
		totalScore += 6
	}

	t, err := time.Parse("15:04", experience.TimeOfBuy)
	if err == nil {
		hour := t.Hour()
		minute := t.Minute()
		if (hour == 14 && minute >= 0) || (hour == 15 && minute < 60) {
			totalScore += 10
		}
	}

	return totalScore
}

func processExperience(w http.ResponseWriter, r *http.Request) {
	var exp ShoppingExperience
	err := json.NewDecoder(r.Body).Decode(&exp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	experienceID := uuid.New().String()
	points := oddTimeCalculator(exp)

	sharedBrain.Lock()
	brain[experienceID] = MemoryBank{
		ShoppingData: exp,
		Score:        points,
	}
	sharedBrain.Unlock()

	response := map[string]string{"id": experienceID}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func fetchScore(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	experienceID := vars["id"]

	sharedBrain.Lock()
	data, found := brain[experienceID]
	sharedBrain.Unlock()

	if !found {
		http.Error(w, "No clue where that receipt is.", http.StatusNotFound)
		return
	}

	response := map[string]int{"points": data.Score}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func main() {
	paths := mux.NewRouter()
	paths.HandleFunc("/receipts/process", processExperience).Methods("POST")
	paths.HandleFunc("/receipts/{id}/points", fetchScore).Methods("GET")

	fmt.Println("server starting on port 8080...")
	http.ListenAndServe(":8080", paths)
}
