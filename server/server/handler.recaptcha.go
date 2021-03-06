package server

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/spf13/viper"
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

const RECAPTCHA_API_BASEURL = "https://www.google.com/recaptcha/api/siteverify"

func check(recaptchaPrivateKey, response string) (recaptchaResponse, error) {
	var r recaptchaResponse
	resp, err := http.PostForm(RECAPTCHA_API_BASEURL,
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
		result, err := check(viper.GetString("recaptchaPrivateKey"), request.RecaptchaToken)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		json.NewEncoder(w).Encode(result)
	}
}
