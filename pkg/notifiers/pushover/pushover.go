package pushover

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

const endpoint string = "https://api.pushover.net/1/messages.json"

type Priority int

const (
	PriorityNoNotification      Priority = -2
	PrioritySilent              Priority = -1
	PriorityNormal              Priority = 0
	PriorityHighPriority        Priority = 1
	PriorityRequireConfirmation Priority = 2
)

type Response struct {
	Status  int    `json:"status"`
	Request string `json:"request"`
}

type Pushover struct {
	ApiToken string
	Users    []string
}

func New(apiToken string, users []string) *Pushover {
	return &Pushover{
		ApiToken: apiToken,
		Users:    users,
	}
}

func (p *Pushover) Notify(bins []string) error {
	msg := fmt.Sprintf("Bins out today: %s", strings.Join(bins, ", "))

	for _, user := range p.Users {
		payload := url.Values{}
		payload.Add("token", p.ApiToken)
		payload.Add("user", user)
		payload.Add("priority", strconv.Itoa(int(PriorityNormal)))
		payload.Add("timestamp", strconv.Itoa(int(time.Now().Unix())))
		payload.Add("message", msg)
		if err := p.send(payload); err != nil {
			return err
		}
		log.WithField("user", user[:8]+"...").Info("pushover: notification sent")
	}
	return nil
}

func (p *Pushover) send(payload url.Values) error {
	resp, err := http.PostForm(endpoint, payload)
	if err != nil {
		return fmt.Errorf("pushover: error sending notification: %s", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("pushover: unable to read response body: %s", err)
	}

	if resp.StatusCode > 200 && resp.StatusCode < 500 {
		return fmt.Errorf("pushover: bad status code %d from Pushover API, response body: %s",
			resp.StatusCode, body)
	}

	if resp.StatusCode >= 500 {
		return fmt.Errorf("pushover: status code %d from Pushover API indicates temporary failure, but not retrying", resp.StatusCode)
	}

	pr := Response{}
	err = json.Unmarshal(body, &pr)
	if err != nil {
		log.WithField("body", body).Error("pushover: failed to unmarshal response from API")
		return fmt.Errorf("pushover: could not unmarshal response from Pushover API: %s", err)
	}

	if pr.Status != 1 {
		return fmt.Errorf("pushover: API status was %d, expected 1, notification may not have been sent", pr.Status)
	}

	return nil
}
