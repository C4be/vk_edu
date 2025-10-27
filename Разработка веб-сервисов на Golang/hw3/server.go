package main

import (
	"encoding/json"
	"encoding/xml"
	"io"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
)

var datasetFile = "/mnt/data/dataset.xml" // можно подменять из тестов

type xmlRoot struct {
	Rows []xmlRow `xml:"row"`
}

type xmlRow struct {
	ID        int    `xml:"id"`
	FirstName string `xml:"first_name"`
	LastName  string `xml:"last_name"`
	Age       int    `xml:"age"`
	About     string `xml:"about"`
	Gender    string `xml:"gender"`
}

type SearchErrResp struct {
	Error string `json:"error"`
}

func sendError(w http.ResponseWriter, status int, msg string) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(SearchErrResp{Error: msg}) // nolint:errcheck
}

// SearchServer HTTP handler
func SearchServer(w http.ResponseWriter, r *http.Request) {
	// Authorization: require AccessToken header non-empty
	if r.Header.Get("AccessToken") == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// read parameters
	q := r.FormValue("query")
	orderField := r.FormValue("order_field")
	orderByStr := r.FormValue("order_by")
	limitStr := r.FormValue("limit")
	offsetStr := r.FormValue("offset")

	// defaults and parsing
	limit := 0
	offset := 0
	orderBy := 0
	var err error
	if limitStr != "" {
		limit, err = strconv.Atoi(limitStr)
		if err != nil {
			// bad request
			sendError(w, http.StatusBadRequest, "limit param invalid")
			return
		}
	}
	if offsetStr != "" {
		offset, err = strconv.Atoi(offsetStr)
		if err != nil {
			sendError(w, http.StatusBadRequest, "offset param invalid")
			return
		}
	}
	if orderByStr != "" {
		orderBy, err = strconv.Atoi(orderByStr)
		if err != nil {
			sendError(w, http.StatusBadRequest, "order_by param invalid")
			return
		}
	}

	// if order_field empty -> default to Name
	if orderField == "" {
		orderField = "Name"
	}

	// accept case-insensitive fields but match allowed set
	of := strings.ToLower(orderField)
	if of != "id" && of != "age" && of != "name" {
		sendError(w, http.StatusBadRequest, ErrorBadOrderField)
		return
	}

	// open dataset
	f, err := os.Open(datasetFile)
	if err != nil {
		// internal server error
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer f.Close()

	// parse XML
	content, err := io.ReadAll(f)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var root xmlRoot
	if err = xml.Unmarshal(content, &root); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// convert to User slice (same struct as client.User)
	users := make([]User, 0, len(root.Rows))
	for _, rr := range root.Rows {
		u := User{
			ID:     rr.ID,
			Name:   strings.TrimSpace(rr.FirstName + " " + rr.LastName),
			Age:    rr.Age,
			About:  rr.About,
			Gender: rr.Gender,
		}
		// filter by query if provided (substring in Name or About)
		if q == "" || strings.Contains(u.Name, q) || strings.Contains(u.About, q) {
			users = append(users, u)
		}
	}

	// ordering
	switch of {
	case "name":
		if orderBy == OrderByAsIs {
			// as is - do nothing
		} else {
			sort.SliceStable(users, func(i, j int) bool {
				if orderBy == OrderByAsc {
					return users[i].Name < users[j].Name
				}
				// desc
				return users[i].Name > users[j].Name
			})
		}
	case "id":
		if orderBy == OrderByAsIs {
		} else {
			sort.SliceStable(users, func(i, j int) bool {
				if orderBy == OrderByAsc {
					return users[i].ID < users[j].ID
				}
				return users[i].ID > users[j].ID
			})
		}
	case "age":
		if orderBy == OrderByAsIs {
		} else {
			sort.SliceStable(users, func(i, j int) bool {
				if orderBy == OrderByAsc {
					return users[i].Age < users[j].Age
				}
				return users[i].Age > users[j].Age
			})
		}
	}

	// offset/limit slicing
	// protect bounds
	if offset < 0 {
		offset = 0
	}
	start := offset
	if start > len(users) {
		start = len(users)
	}
	end := len(users)
	if limit > 0 {
		end = start + limit
		if end > len(users) {
			end = len(users)
		}
	}
	out := users[start:end]

	// respond with JSON array
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	_ = enc.Encode(out) // nolint:errcheck
}
