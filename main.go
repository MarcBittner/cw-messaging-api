package main

import (
	"encoding/json"
	"net/http"
	"os"
	"github.com/twilio/twilio-go"
//	"github.com/twilio/twilio-go/client"
	openapi "github.com/twilio/twilio-go/rest/api/v2010"
)

type MessageRequest struct {
	To      string `json:"to"`
	Body    string `json:"body"`
	Subject string `json:"subject,omitempty"` // Optional, for LinkedIn
}

func main() {
	http.HandleFunc("/send/sms", sendSMSMessage)
	http.HandleFunc("/send/linkedin", sendLinkedInMessage)

	http.ListenAndServe(":8080", nil)
}

func sendSMSMessage(w http.ResponseWriter, r *http.Request) {
	var request MessageRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	client := twilio.NewRestClientWithParams(twilio.ClientParams{
		Username: os.Getenv("TWILIO_ACCOUNT_SID"),
		Password: os.Getenv("TWILIO_AUTH_TOKEN"),
	})

	// params := &openapi.CreateMessageParams{}
	// params.SetTo(request.To)
	// params.SetFrom(os.Getenv("TWILIO_PHONE_NUMBER"))
	// params.SetBody(request.Body)
	params := &openapi.CreateMessageParams{}
	params.SetTo(os.Getenv("TO_PHONE_NUMBER"))
	params.SetFrom(os.Getenv("TWILIO_PHONE_NUMBER"))
	params.SetBody("Hello from Golang!")

	// _, err := client.ApiV2010.CreateMessage(params)
	_, err := client.Api.CreateMessage(params)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "SMS message sent"})
}

func sendLinkedInMessage(w http.ResponseWriter, r *http.Request) {
	var request MessageRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Placeholder for LinkedIn message sending logic
	// This typically involves OAuth2 authentication and making a POST request to LinkedIn's API
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{"status": "LinkedIn message sending not implemented"})
}
