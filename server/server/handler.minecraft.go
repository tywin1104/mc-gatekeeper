package server

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type uuid struct {
	UUID string `json:"id"`
}

type profile struct {
	Properties []property `json:"properties"`
}

type property struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type value struct {
	Textures texture `json:"textures"`
}
type texture struct {
	Skin skin `json:"SKIN"`
}
type skin struct {
	URL string `json:"url"`
}

// RateLimitError is returned when access to Majong API is denied dur to rate limiting
type RateLimitError struct {
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("API rate limit reached. Try later")
}

// PropertyNotFoundError is returned when required property field does not exist
type PropertyNotFoundError struct {
	field string
}

func (e *PropertyNotFoundError) Error() string {
	return fmt.Sprintf("Required property %s does not exist", e.field)
}

func getUUID(username string) (string, error) {
	u := url.URL{
		Scheme: "https",
		Host:   "api.mojang.com",
		Path:   "users/profiles/minecraft/" + username,
	}

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return "", err
	}
	// Add user-agent to prevent cloudfront 403 response
	req.Header.Set("User-Agent", "minecraft")
	var uid uuid
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusTooManyRequests {
			err = &RateLimitError{}
		} else {
			err = fmt.Errorf("status code: %d", resp.StatusCode)
		}
		return "", err
	}
	defer resp.Body.Close()
	err = json.NewDecoder(resp.Body).Decode(&uid)
	if err != nil {
		return "", err
	}
	return uid.UUID, nil
}

func getSkinBase64FromProfile(uuid string) (string, error) {
	u := url.URL{
		Scheme: "https",
		Host:   "sessionserver.mojang.com",
		Path:   "session/minecraft/profile/" + uuid,
	}

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("User-Agent", "minecraft")
	var profile profile
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusTooManyRequests {
			err = &RateLimitError{}
		} else {
			err = fmt.Errorf("status code: %d", resp.StatusCode)
		}
		return "", err
	}
	defer resp.Body.Close()
	err = json.NewDecoder(resp.Body).Decode(&profile)
	if err != nil {
		return "", err
	}
	// Search for textures(skin) field
	for _, prop := range profile.Properties {
		if prop.Name == "textures" {
			return prop.Value, nil
		}
	}
	return "", &PropertyNotFoundError{field: "textures"}
}

func getSkinURL(encodedVal string) (string, error) {
	sDec, err := base64.StdEncoding.DecodeString(encodedVal)
	if err != nil {
		return "", err
	}
	var v value
	json.Unmarshal(sDec, &v)
	return v.Textures.Skin.URL, nil
}

func (svc *Service) handleGetSkinURLByUsername() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username := mux.Vars(r)["minecraftUsername"]
		uuid, err := getUUID(username)
		if err != nil {
			fmt.Println(err)
		}
		val, err := getSkinBase64FromProfile(uuid)
		if err != nil {
			switch err.(type) {
			case *RateLimitError:
				w.WriteHeader(http.StatusTooManyRequests)
				return
			}
			svc.logger.WithFields(logrus.Fields{
				"error": err.Error(),
			}).Error("Unable to get texture for the user")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		url, err := getSkinURL(val)
		if err != nil {
			svc.logger.WithFields(logrus.Fields{
				"error": err.Error(),
			}).Error("Unable to get skin url from the encoded texture value")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		msg := map[string]map[string]interface{}{"skin": {
			"url": url,
		}}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(msg)
	}
}
