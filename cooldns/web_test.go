package cooldns

import (
	"testing"
	"log"
	"net/http"
	"net/http/httptest"
	"bytes"
)

func createTestServer(t *testing.T) (*httptest.Server, *bytes.Buffer) {
	f, err := getTmpFile()
	if err != nil {
		t.Error("setup Failed: Could not create test server")
	}

	// start Server TODO: make stopable
	db, err := getDB(f)
	if err != nil {
		t.Error("Failed to create temporary DB:", err)
	}
	// Setup logger to log to our buffer, print on failure later
	logBuf := new(bytes.Buffer)
	log.SetOutput(logBuf)
	log.Println("##### TESTING LOG BUFFER #####")

	handler := SetupWeb(db, "../assets", "../templates")
	return httptest.NewServer(handler), logBuf
}

func TestGetIndex(t *testing.T) {
	server, logBuf := createTestServer(t)
	defer server.Close()

	resp, err := http.Get(server.URL+"/")
	if err != nil {
		t.Log(logBuf.String())
		t.Error("Failed to connect to server")
	}
	if resp.StatusCode != 200 {
		t.Log(logBuf.String())
		t.Error("Status for \"/\" is not 200")
	}
}
