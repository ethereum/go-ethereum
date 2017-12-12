package notifications

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/status-go/geth/params"
)

// NotificationDeliveryProvider handles the notification delivery
type NotificationDeliveryProvider interface {
	Send(id string, payload string) error
}

// FirebaseProvider represents FCM provider
type FirebaseProvider struct {
	AuthorizationKey       string
	NotificationTriggerURL string
}

// NewFirebaseProvider creates new FCM provider
func NewFirebaseProvider(config *params.FirebaseConfig) *FirebaseProvider {
	authorizationKey, _ := config.ReadAuthorizationKeyFile()
	return &FirebaseProvider{
		NotificationTriggerURL: config.NotificationTriggerURL,
		AuthorizationKey:       string(authorizationKey),
	}
}

// Send triggers sending of Push Notification to a given device id
func (p *FirebaseProvider) Send(id string, payload string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()

	jsonRequest := strings.Replace(payload, "{{ ID }}", id, 3)
	req, err := http.NewRequest("POST", p.NotificationTriggerURL, bytes.NewBuffer([]byte(jsonRequest)))
	req.Header.Set("Authorization", "key="+p.AuthorizationKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	log.Debug("FCM response", "status", resp.Status, "header", resp.Header)
	body, _ := ioutil.ReadAll(resp.Body)
	log.Debug("FCM response body", "body", string(body))

	return nil
}
