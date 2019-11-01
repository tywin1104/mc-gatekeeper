package server

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
)

type recaptchaRequest struct {
	RecaptchaToken string `json:"recapchaToken"`
}

type recaptchaResponse struct {
	Success     bool      `json:"success"`
	ChallengeTS time.Time `json:"challenge_ts"`
	Hostname    string    `json:"hostname"`
	ErrorCodes  []string  `json:"error-codes"`
}

const recaptchaServerName = "https://www.google.com/recaptcha/api/siteverify"

func check(recaptchaPrivateKey, response string) (recaptchaResponse, error) {
	var r recaptchaResponse
	resp, err := http.PostForm(recaptchaServerName,
		url.Values{"secret": {recaptchaPrivateKey}, "response": {response}})
	if err != nil {
		return recaptchaResponse{}, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return recaptchaResponse{}, err
	}
	err = json.Unmarshal(body, &r)
	if err != nil {
		return recaptchaResponse{}, err
	}
	return r, nil
}

func (svc *Service) handleVerifyRecaptcha() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqBody, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Unable to read request body", http.StatusInternalServerError)
			return
		}
		var request recaptchaRequest
		err = json.Unmarshal(reqBody, &request)
		if err != nil {
			http.Error(w, "Unable to unmarshal request body", http.StatusInternalServerError)
			return
		}
		result, err := check(svc.c.RecaptchaPrivateKey, request.RecaptchaToken)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		json.NewEncoder(w).Encode(result)
	}
}
