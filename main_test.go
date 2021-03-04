package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func init()  {
	// Force createID to always create an ID of 1 so we can test easier
	createID = func() string {
		return "1"
	}
}


func Test_getClasses(t *testing.T) {
	t.Run("Get classes when their is zero classes", func(t *testing.T) {
		// get fake reader and writer for request
		r, _ := http.NewRequest("GET", "/classes", nil)
		w := httptest.NewRecorder()

		getClasses(w, r)
		var response []map[string]string
		respBody, _ := ioutil.ReadAll(w.Body)
		json.Unmarshal(respBody, &response)
		assert.Equal(t, len(response), 0)
		assert.Equal(t, http.StatusOK, w.Code)
	})
	t.Run("Get classes, when their is two classes", func(t *testing.T) {
		// get fake reader and writer for request
		r, _ := http.NewRequest("GET", "/classes", nil)
		w := httptest.NewRecorder()

		DBClasses = []Class{
			{
				Id:       "1",
				Name:     "class 1",
				Date:     time.Date(2020, 12, 12, 0, 0, 0, 0, time.UTC),
				Capacity: 20,
				Bookings: []Booking{{MemberName: "David"}},
			},
			{
				Id:       "2",
				Name:     "class 2",
				Date:     time.Date(2020, 12, 13, 0, 0, 0, 0, time.UTC),
				Capacity: 10,
				Bookings: []Booking{},
			},
		}
		expectedResponse := `[{"id":"1","name":"class 1","date":"2020-12-12T00:00:00Z","capacity":20},` +
			 				 `{"id":"2","name":"class 2","date":"2020-12-13T00:00:00Z","capacity":10}]` + "\n"
		getClasses(w, r)
		respBody, _ := ioutil.ReadAll(w.Body)

		assert.Equal(t, expectedResponse, string(respBody))
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func Test_createClass(t *testing.T) {
	t.Run("Create a single class", func(t *testing.T) {
		DBClasses = []Class{}
		// get fake reader and writer for request
		body := []byte(`{"name": "kayak","start_date": "2006-01-01","end_date": "2006-01-01", "capacity": 20}`)
		r, _ := http.NewRequest("POST", "/classes", bytes.NewReader(body))
		w := httptest.NewRecorder()

		createClass(w, r)
		var response []Class
		respBody, _ := ioutil.ReadAll(w.Body)

		expectedDate, _ := time.Parse(layoutISO, "2006-01-01")
		json.Unmarshal(respBody, &response)
		assert.Equal(t, "kayak", response[0].Name)
		assert.Equal(t, 20, response[0].Capacity)
		assert.Equal(t, expectedDate, response[0].Date)
		assert.Equal(t, http.StatusCreated, w.Code)
	})
	t.Run("Create a class spanning 5 days", func(t *testing.T) {
		DBClasses = []Class{}

		body := []byte(`{"name": "kayak","start_date": "2006-01-01","end_date": "2006-01-05", "capacity": 20}`)
		expectedStartDate, _ := time.Parse(layoutISO, "2006-01-01")
		r, _ := http.NewRequest("POST", "/classes", bytes.NewReader(body))
		w := httptest.NewRecorder()

		createClass(w, r)
		var response []Class
		respBody, _ := ioutil.ReadAll(w.Body)
		json.Unmarshal(respBody, &response)

		assert.Equal(t, "kayak", response[0].Name)
		assert.Equal(t, 20, response[0].Capacity)
		assert.Equal(t, 5, len(response))
		assert.Equal(t, expectedStartDate, response[0].Date)
		assert.Equal(t, http.StatusCreated, w.Code)
	})
	t.Run("try create class with malformed json request", func(t *testing.T) {
		DBClasses = []Class{}

		body := []byte(`{"name": "kayak","start_date": "2006-01-01","end_date": "2006-01-05" "capacity": 20}`)
		r, _ := http.NewRequest("POST", "/classes", bytes.NewReader(body))
		w := httptest.NewRecorder()

		createClass(w, r)
		var errorResponse ErrorResponse
		respBody, _ := ioutil.ReadAll(w.Body)
		json.Unmarshal(respBody, &errorResponse)

		assert.Equal(t, InvalidJSON, errorResponse.Err)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
	t.Run("try create class with malformed start date request", func(t *testing.T) {
		DBClasses = []Class{}

		body := []byte(`{"name": "kayak","start_date": "2006-13-12","end_date": "2006-01-05", "capacity": 20}`)
		r, _ := http.NewRequest("POST", "/classes", bytes.NewReader(body))
		w := httptest.NewRecorder()

		createClass(w, r)
		var errorResponse ErrorResponse
		respBody, _ := ioutil.ReadAll(w.Body)
		json.Unmarshal(respBody, &errorResponse)

		assert.Equal(t, InvalidDate, errorResponse.Err)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func Test_createBooking(t *testing.T) {
	t.Run("create a booking", func(t *testing.T) {
		//Adding a class to are pretend DB
		DBClasses = []Class{
			{
				Id:       "1",
				Name:     "lifting",
				Date:     time.Date(2020, 12, 12, 0, 0, 0, 0, time.UTC),
				Capacity: 20,
				Bookings: nil,
			},
		}

		requestBody := []byte(`{"member_name":"David","class_name":"lifting","date":"2020-12-12"}` + "\n")
		r, _ := http.NewRequest("POST", "/classes", bytes.NewReader(requestBody))
		w := httptest.NewRecorder()

		createBooking(w, r)
		expectedRespBody := []byte(`{"id":"1","member_name":"David","class_name":"lifting","date":"2020-12-12"}` + "\n")
		respBody, _ := ioutil.ReadAll(w.Body)
		assert.Equal(t, string(expectedRespBody), string(respBody))
		//Make sure the booking is properly append to the correct Class in DBClasses
		assert.Equal(t, Booking{MemberName: "David", Id: "1"}, DBClasses[0].Bookings[0])
		assert.Equal(t, http.StatusCreated, w.Code)
	})
	t.Run("try create a booking for a class that doesn't exist", func(t *testing.T) {
		DBClasses = []Class{}

		body := []byte(`{"member_name": "David","class_name": "lifting","date": "2020-12-12"}`)
		r, _ := http.NewRequest("POST", "/classes", bytes.NewReader(body))
		w := httptest.NewRecorder()

		createBooking(w, r)

		var errorResponse ErrorResponse
		respBody, _ := ioutil.ReadAll(w.Body)
		json.Unmarshal(respBody, &errorResponse)

		assert.Equal(t, ClassDoesNotExists, errorResponse.Err)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
	t.Run("try create a booking malformed json request", func(t *testing.T) {
		DBClasses = []Class{}

		body := []byte(`{"member_na "David","class_name": "lifting","date": "2020-12-12"}`)
		r, _ := http.NewRequest("POST", "/classes", bytes.NewReader(body))
		w := httptest.NewRecorder()

		createBooking(w, r)

		var errorResponse ErrorResponse
		respBody, _ := ioutil.ReadAll(w.Body)
		json.Unmarshal(respBody, &errorResponse)

		assert.Equal(t, InvalidJSON, errorResponse.Err)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
	t.Run("try create a booking with a malformed date request", func(t *testing.T) {
		DBClasses = []Class{}

		body := []byte(`{"member_name": "David","class_name": "lifting","date": "2020-12-11222222222222222"}`)
		r, _ := http.NewRequest("POST", "/classes", bytes.NewReader(body))
		w := httptest.NewRecorder()

		createBooking(w, r)

		var errorResponse ErrorResponse
		respBody, _ := ioutil.ReadAll(w.Body)
		json.Unmarshal(respBody, &errorResponse)

		assert.Equal(t, InvalidDate, errorResponse.Err)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func Test_errorResponse(t *testing.T) {
	t.Run("test error message and response code are correct", func(t *testing.T) {
		w := httptest.NewRecorder()

		givenReason := "reason a"
		httpErrorCode := http.StatusTeapot
		errorResponse(w, givenReason, httpErrorCode)

		var errorResponse ErrorResponse
		respBody, _ := ioutil.ReadAll(w.Body)
		json.Unmarshal(respBody, &errorResponse)

		assert.Equal(t, givenReason, errorResponse.Err)
		assert.Equal(t, httpErrorCode, w.Code)
	})
}

func Test_getClass(t *testing.T) {
	t.Run("malformed date request", func(t *testing.T) {
		w := httptest.NewRecorder()

		givenReason := "reason a"
		httpErrorCode := http.StatusTeapot
		errorResponse(w, givenReason, httpErrorCode)

		var errorResponse ErrorResponse
		respBody, _ := ioutil.ReadAll(w.Body)
		json.Unmarshal(respBody, &errorResponse)

		assert.Equal(t, givenReason, errorResponse.Err)
		assert.Equal(t, httpErrorCode, w.Code)
	})
}
