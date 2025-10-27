package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFindUsers_Success_MaxAgeExample(t *testing.T) {
	// start test server with our handler
	datasetFile = "dataset.xml"
	srv := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer srv.Close()

	client := &SearchClient{
		AccessToken: "token",
		URL:         srv.URL,
	}

	// Example in task: order_by=-1&order_field=age&limit=1&offset=0&query=on
	req := SearchRequest{
		Limit:      1,
		Offset:     0,
		Query:      "on",
		OrderField: "age",
		OrderBy:    -1,
	}

	resp, err := client.FindUsers(req)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if resp == nil {
		t.Fatalf("nil response")
	}
	// should return single user (client increments limit internally)
	if len(resp.Users) != 1 {
		t.Fatalf("expected 1 user, got %d", len(resp.Users))
	}
	// From dataset the user with max age among matches has ID 13
	if resp.Users[0].ID != 13 {
		t.Fatalf("expected user id 13, got %d", resp.Users[0].ID)
	}
}

func TestFindUsers_Unauthorized(t *testing.T) {
	datasetFile = "dataset.xml"
	srv := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer srv.Close()

	client := &SearchClient{
		AccessToken: "", // missing token -> 401
		URL:         srv.URL,
	}
	req := SearchRequest{
		Limit:  1,
		Offset: 0,
	}
	_, err := client.FindUsers(req)
	if err == nil {
		t.Fatalf("expected error for unauthorized, got nil")
	}
	if !strings.Contains(err.Error(), "bad AccessToken") {
		t.Fatalf("expected bad AccessToken error, got: %v", err)
	}
}

func TestFindUsers_BadOrderField(t *testing.T) {
	datasetFile = "dataset.xml"
	srv := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer srv.Close()

	client := &SearchClient{
		AccessToken: "token",
		URL:         srv.URL,
	}
	req := SearchRequest{
		Limit:      2,
		Offset:     0,
		Query:      "a",
		OrderField: "NotAField",
		OrderBy:    1,
	}
	_, err := client.FindUsers(req)
	if err == nil {
		t.Fatalf("expected error for bad order field")
	}
	// client formats this into "OrderFeld %s invalid"
	if !strings.Contains(err.Error(), "OrderFeld") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestFindUsers_PaginationAndOrderByNameAsc(t *testing.T) {
	datasetFile = "dataset.xml"
	srv := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer srv.Close()

	c := &SearchClient{
		AccessToken: "token",
		URL:         srv.URL,
	}

	// request first 2 users ordered by Name ascending
	req := SearchRequest{
		Limit:      2,
		Offset:     0,
		Query:      "",
		OrderField: "Name",
		OrderBy:    1,
	}
	resp, err := c.FindUsers(req)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(resp.Users) != 2 {
		t.Fatalf("expected 2 users, got %d", len(resp.Users))
	}
	// verify they are the first two in name-sorted order by reading dataset and checking names
	// we will simply check IDs that are known from dataset sorted order
	wantFirst := 15 // Allison Valdez
	wantSecond := 16 // Annie Osborn
	if resp.Users[0].ID != wantFirst || resp.Users[1].ID != wantSecond {
		t.Fatalf("unexpected ids: got %v, want %v,%v", []int{resp.Users[0].ID, resp.Users[1].ID}, wantFirst, wantSecond)
	}
}

func TestSearchServer_InternalErrorOnMissingFile(t *testing.T) {
	// point to non-existent file to cause 500
	datasetFile = "/nonexistent/file.xml"
	srv := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer srv.Close()

	c := &SearchClient{
		AccessToken: "token",
		URL:         srv.URL,
	}
	req := SearchRequest{
		Limit:  1,
		Offset: 0,
	}
	_, err := c.FindUsers(req)
	if err == nil {
		t.Fatalf("expected server fatal error")
	}
	if !strings.Contains(err.Error(), "SearchServer fatal error") {
		t.Fatalf("expected 'SearchServer fatal error', got: %v", err)
	}
}

// Additional small test to ensure server returns valid JSON array shape that client can parse
func TestSearchServer_JSONStructure(t *testing.T) {
	datasetFile = "dataset.xml"
	srv := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer srv.Close()
	// call server directly to get raw body and ensure valid JSON array
	url := srv.URL + "?limit=1&offset=0&order_field=Name&order_by=0"
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("AccessToken", "token")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("http do error: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 got %d", res.StatusCode)
	}
	var arr []User
	err = json.NewDecoder(res.Body).Decode(&arr)
	if err != nil {
		t.Fatalf("json decode failed: %v", err)
	}
	// check that decoding produced either 0 or more users and fields are present
	for _, u := range arr {
		if u.ID < 0 {
			t.Fatalf("invalid id %d", u.ID)
		}
		if u.Name == "" {
			t.Fatalf("empty name")
		}
		if u.Age < 0 {
			t.Fatalf("invalid age %d", u.Age)
		}
	}
}

func TestSearchServer_InvalidParams(t *testing.T) {
	datasetFile = "dataset.xml"
	srv := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer srv.Close()

	tests := []string{
		"?limit=abc",             // bad limit
		"?offset=zzz",            // bad offset
		"?order_by=oops",         // bad order_by
		"?order_field=wrongfield", // bad order field
	}
	for _, qs := range tests {
		req, _ := http.NewRequest("GET", srv.URL+qs, nil)
		req.Header.Add("AccessToken", "token")
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if res.StatusCode != http.StatusBadRequest {
			t.Errorf("expected 400 for %s, got %d", qs, res.StatusCode)
		}
		res.Body.Close()
	}
}

func TestSearchServer_OrderByAsIs_NoSort(t *testing.T) {
	datasetFile = "dataset.xml"
	srv := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer srv.Close()

	req, _ := http.NewRequest("GET", srv.URL+"?limit=3&order_field=age&order_by=0", nil)
	req.Header.Add("AccessToken", "token")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 got %d", resp.StatusCode)
	}

	var users []User
	if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if len(users) == 0 {
		t.Fatalf("expected non-empty list")
	}
}

func TestSearchServer_EmptyQuery_NoMatches(t *testing.T) {
	datasetFile = "dataset.xml"
	srv := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer srv.Close()

	req, _ := http.NewRequest("GET", srv.URL+"?query=ZZZZZZZ", nil)
	req.Header.Add("AccessToken", "token")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("http error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 got %d", resp.StatusCode)
	}
	var users []User
	json.NewDecoder(resp.Body).Decode(&users)
	if len(users) != 0 {
		t.Fatalf("expected 0 matches, got %d", len(users))
	}
}

func TestSearchServer_OffsetBeyondRange(t *testing.T) {
	datasetFile = "dataset.xml"
	srv := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer srv.Close()

	req, _ := http.NewRequest("GET", srv.URL+"?offset=9999", nil)
	req.Header.Add("AccessToken", "token")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 got %d", resp.StatusCode)
	}
	var users []User
	json.NewDecoder(resp.Body).Decode(&users)
	if len(users) != 0 {
		t.Fatalf("expected 0 users, got %d", len(users))
	}
}


func TestClient_NegativeLimitOffset(t *testing.T) {
	c := &SearchClient{AccessToken: "token", URL: "http://example.com"}
	_, err := c.FindUsers(SearchRequest{Limit: -1})
	if err == nil || !strings.Contains(err.Error(), "limit must be > 0") {
		t.Fatalf("expected limit error, got %v", err)
	}
	_, err = c.FindUsers(SearchRequest{Offset: -5})
	if err == nil || !strings.Contains(err.Error(), "offset must be > 0") {
		t.Fatalf("expected offset error, got %v", err)
	}
}

func TestClient_LimitTooLarge(t *testing.T) {
	// Проверяем, что limit > 25 урезается до 25 и nextPage выставляется корректно
	datasetFile = "dataset.xml"
	srv := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer srv.Close()

	client := &SearchClient{
		AccessToken: "token",
		URL:         srv.URL,
	}

	req := SearchRequest{
		Limit:      100, // больше 25
		Offset:     0,
		OrderField: "id",
		OrderBy:    1,
	}
	resp, err := client.FindUsers(req)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !resp.NextPage && len(resp.Users) >= 25 {
		t.Fatalf("expected NextPage true for truncated limit")
	}
}

func TestClient_BadJSONErrorResponse(t *testing.T) {
	// создаем сервер, который вернет битый JSON при 400
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("{not valid json")) // битый JSON
	}))
	defer srv.Close()

	client := &SearchClient{
		AccessToken: "token",
		URL:         srv.URL,
	}

	req := SearchRequest{Limit: 1, Offset: 0}
	_, err := client.FindUsers(req)
	if err == nil || !strings.Contains(err.Error(), "cant unpack error json") {
		t.Fatalf("expected cant unpack error json, got %v", err)
	}
}

func TestClient_BadJSONResult(t *testing.T) {
	// сервер возвращает 200, но невалидный JSON
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{invalid_json"))
	}))
	defer srv.Close()

	client := &SearchClient{
		AccessToken: "token",
		URL:         srv.URL,
	}

	req := SearchRequest{Limit: 1, Offset: 0}
	_, err := client.FindUsers(req)
	if err == nil || !strings.Contains(err.Error(), "cant unpack result json") {
		t.Fatalf("expected cant unpack result json error, got %v", err)
	}
}

func TestClient_NextPageFalse(t *testing.T) {
	datasetFile = "dataset.xml"
	srv := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer srv.Close()

	client := &SearchClient{AccessToken: "token", URL: srv.URL}

	req := SearchRequest{
		Limit:      5,
		Offset:     45, // ближе к концу, где мало записей
		OrderField: "id",
		OrderBy:    1,
	}
	resp, err := client.FindUsers(req)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if resp.NextPage {
		t.Fatalf("expected NextPage false near end, got true")
	}
}

