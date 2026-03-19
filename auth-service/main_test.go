package main

import (
	"bytes"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLoginHandler_Unauthorized(t *testing.T) {
	oldQueryUserID := queryUserID
	t.Cleanup(func() { queryUserID = oldQueryUserID })
	queryUserID = func(username, password string) (int, error) {
		return 0, sql.ErrNoRows
	}

	req, _ := http.NewRequest("POST", "/login", bytes.NewBuffer([]byte(`{"username":"x","password":"y"}`)))
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(loginHandler)
	
	handler.ServeHTTP(rr, req)
	
	if status := rr.Code; status != http.StatusUnauthorized {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusUnauthorized)
	}
}