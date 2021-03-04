package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

const (
	layoutISO          = "2006-01-02"
	InvalidJSON        = "JSON parse error"
	InternalError      = "Internal error please try again"
	InvalidDate        = "Could not parse date, format should be YYYY-MM-DD"
	ClassDoesNotExists = "Requested class does not exist"
)

// instead of reading and writing to a database im just going to keep track of classes in this global slice
var DBClasses = make([]Class, 0)

// findClassReference will return a pointer to the first class with a matching name and date to given input
// in a real real world scenario we'd use its Id to guarantee it was unique
func findClassReference(className string, date time.Time) (*Class, error) {
	for index, class := range DBClasses {
		if class.Name == className && class.Date == date {
			return &DBClasses[index], nil
		}
	}
	return nil, fmt.Errorf("that class does not exsist")
}

type Booking struct {
	MemberName string
	Id         string
}

type BookingRequest struct {
	Id         string `json:"id"`
	MemberName string `json:"member_name"`
	ClassName  string `json:"class_name"`
	Date       string `json:"date"`
}

type Class struct {
	Id       string    `json:"id"`
	Name     string    `json:"name"`
	Date     time.Time `json:"date"`
	Capacity int       `json:"capacity"`
	Bookings []Booking `json:"-"`
}

func (class *Class) addBooking(booking Booking) {
	class.Bookings = append(class.Bookings, booking)
}

type ClassRequest struct {
	Name      string `json:"name"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
	Capacity  int    `json:"capacity"`
}

// createID creates a unique id
var createID = func() string{
	return uuid.New().String()
}

type ErrorResponse struct {
	Err string `json:"error"`
}

// errorResponse will write an error json constructed from inputs to ResponseWriter
func errorResponse(w http.ResponseWriter, reason string, statusCode int) error {
	w.WriteHeader(statusCode)
	errResponse := ErrorResponse{Err: reason}
	err := json.NewEncoder(w).Encode(errResponse)
	if err != nil {
		return err
	}
	return nil
}

// createClass is the handler function for POST requests to `/classes`, it will parse the request body, validate it and
// append classes to `DBClasses`. Will append 1 class for each day in the range from start_date to end_date
func createClass(w http.ResponseWriter, r *http.Request) {
	reqBody, _ := ioutil.ReadAll(r.Body)

	var classRequest ClassRequest
	err := json.Unmarshal(reqBody, &classRequest)
	if err != nil {
		err = errorResponse(w, InvalidJSON, http.StatusBadRequest)
		if err != nil {
			fmt.Println(err)
		}
		return
	}

	var classes []Class
	startDate, err := time.Parse(layoutISO, classRequest.StartDate)
	if err != nil {
		err = errorResponse(w, InvalidDate, http.StatusBadRequest)
		if err != nil {
			fmt.Println(err)
		}
		return
	}
	endDate, err := time.Parse(layoutISO, classRequest.EndDate)
	if err != nil {
		err = errorResponse(w, InvalidDate, http.StatusBadRequest)
		if err != nil {
			fmt.Println(err)
		}
		return
	}

	for days := 0; days <= int(endDate.Sub(startDate).Hours()/24); days++ {
		class := Class{
			Id:       createID(),
			Name:     classRequest.Name,
			Date:     startDate.Add(time.Hour * 24 * time.Duration(days)),
			Capacity: classRequest.Capacity,
		}
		classes = append(classes, class)
	}
	DBClasses = append(DBClasses, classes...)

	w.WriteHeader(http.StatusCreated)
	err = json.NewEncoder(w).Encode(classes)
	if err != nil {
		fmt.Println(err)
		return
	}
}

// getClasses is the handler function for GET requests to `/classes`, it will write to ResponseWriter all classes in `DBClasses`
func getClasses(w http.ResponseWriter, r *http.Request) {
	err := json.NewEncoder(w).Encode(DBClasses)
	if err != nil {
		err = errorResponse(w, InternalError, http.StatusInternalServerError)
		if err != nil {
			fmt.Println(err)
		}
	}
}

// createBooking is the handler function for POST requests to `/bookings`, it will parse the request body, validate it
// and appends a booking to the appropriate class if it exists.
func createBooking(w http.ResponseWriter, r *http.Request) {
	reqBody, _ := ioutil.ReadAll(r.Body)
	var bookingRequest BookingRequest
	err := json.Unmarshal(reqBody, &bookingRequest)
	if err != nil {
		err = errorResponse(w, InvalidJSON, http.StatusBadRequest)
		if err != nil {
			fmt.Println(err)
		}
		return
	}

	date, err := time.Parse(layoutISO, bookingRequest.Date)
	if err != nil {
		err = errorResponse(w, InvalidDate, http.StatusBadRequest)
		if err != nil {
			fmt.Println(err)
		}
		return
	}

	class, err := findClassReference(bookingRequest.ClassName, date)
	if err != nil {
		err = errorResponse(w, ClassDoesNotExists, http.StatusNotFound)
		if err != nil {
			fmt.Println(err)
		}
		return
	}
	bookingRequest.Id = createID()
	class.addBooking(Booking{bookingRequest.MemberName, bookingRequest.Id})
	w.WriteHeader(http.StatusCreated)
	err = json.NewEncoder(w).Encode(bookingRequest)
	if err != nil {
		fmt.Println(err)
	}
}

//  handleRequests handles our request routing
func handleRequests() {
	myRouter := mux.NewRouter().StrictSlash(true)
	myRouter.HandleFunc("/classes", createClass).Methods("POST")
	myRouter.HandleFunc("/classes", getClasses).Methods("GET")
	myRouter.HandleFunc("/bookings", createBooking).Methods("POST")
	log.Fatal(http.ListenAndServe(":10000", myRouter))
}

func main() {
	fmt.Println("Opening Routes:")
	handleRequests()
}
