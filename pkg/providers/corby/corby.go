package corby

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/Xiol/whatbin"
	"github.com/imroc/req/v3"
	log "github.com/sirupsen/logrus"
)

const baseURL string = "https://cms.northnorthants.gov.uk/bin-collection-search/calendarevents"

// Example collectionData:
//
//	{
//	  "id": 25383766,
//	  "title": "Empty CBC Bin Refuse bin 180l",
//	  "subject": "This Premises",
//	  "start": "/Date(1622847599000)/",
//	  "end": null,
//	  "link": "",
//	  "color": "rgb(192,192,192)",
//	  "textColor": "rgb(0,0,0)",
//	  "complete": true,
//	  "url": "#"
//	}
type collectionData struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	Start       string    `json:"start"`
	ParsedStart time.Time `json:"-"`
	End         string    `json:"end"`
	ParsedEnd   time.Time `json:"-"`
	Complete    bool      `json:"complete"`
}

type Provider struct {
	UPRN   string
	client *req.Client
}

func New(uprn string) *Provider {
	return &Provider{
		UPRN: uprn,
		client: req.C().
			SetTimeout(10 * time.Second).
			SetUserAgent("WhatBin/1.0").
			SetBaseURL(baseURL),
	}
}

func (p *Provider) Bins() ([]string, error) {
	log.Info("corby: fetching next collection dates")

	nextCollections, err := p.getNextCollectionData()
	if err != nil {
		return nil, err
	}

	log.WithField("collections", len(nextCollections)).Debug("corby: retrieved next collection dates")

	// We have to put the bins out the night before, so we only care
	// about collections that are taking place tomorrow. If there are
	// no collections for tomorrow then don't do anything.
	tomorrow := time.Now().AddDate(0, 0, 6)
	if len(nextCollections) == 0 || nextCollections[0].ParsedStart.Day() != tomorrow.Day() {
		log.Info("corby: no collections pending")
		return nil, whatbin.ErrNoBinsToday
	}

	// We have collections for tomorrow, so let's figure out the friendly bin
	// names and return them for alerting.
	var bins []string
	for _, collection := range nextCollections {
		bins = append(bins, p.commonBinName(collection.Title))
	}

	log.WithField("bins", bins).Info("corby: bins for collection tomorrow")

	return bins, nil
}

func (p *Provider) getNextCollectionData() ([]*collectionData, error) {
	// We're going to follow the request format that the website uses,
	// which requests the next 7 days of data. Changing the end date
	// doesn't seem to have much effect on the amount of data returned
	// so we'll just roll with it.
	start := time.Now().Format("2006-01-02")
	end := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	var data []*collectionData

	resp, err := p.client.R().
		SetSuccessResult(&data).
		SetPathParams(map[string]string{
			"uprn":  p.UPRN,
			"start": start,
			"end":   end,
		}).Get("/{uprn}/{start}/{end}")

	if err != nil {
		return nil, fmt.Errorf("corby: %s", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("corby: unexpected status code %d", resp.StatusCode)
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("corby: no data returned")
	}

	err = p.parseDates(data)
	if err != nil {
		return nil, err
	}

	return p.filterCollections(data)
}

func (p *Provider) filterCollections(data []*collectionData) ([]*collectionData, error) {
	// The data returned contains past events, and we only care about future ones.
	// Remove any events that are in the past.
	data = slices.DeleteFunc(data, func(cd *collectionData) bool {
		return time.Now().After(cd.ParsedStart)
	})

	// Sort the remaining data by start date
	slices.SortFunc(data, func(i, j *collectionData) int {
		if i.ParsedStart.Before(j.ParsedStart) {
			return -1
		}
		if i.ParsedStart.After(j.ParsedStart) {
			return 1
		}
		return 0
	})

	// Pop the next date off, then delete anything that isn't that date to get
	// the next set of collections.
	nextDate := data[0].ParsedStart
	data = slices.DeleteFunc(data, func(cd *collectionData) bool {
		return cd.ParsedStart != nextDate
	})

	return data, nil
}

func (p *Provider) parseDates(data []*collectionData) error {
	var err error
	for i := range data {
		if data[i].Start != "" {
			data[i].ParsedStart, err = p.convertDate(data[i].Start)
			if err != nil {
				log.WithError(err).WithField("start", data[i].Start).Error("corby: failed to convert start date")
			}
		}

		if data[i].End != "" {
			data[i].ParsedEnd, err = p.convertDate(data[i].End)
			if err != nil {
				log.WithError(err).WithField("end", data[i].End).Error("corby: failed to convert end date")
			}
		}
	}

	return nil
}

func (p *Provider) convertDate(date string) (time.Time, error) {
	// The date format is "/Date(1612348800000)/", we need to remove the rubbish
	// and convert the Unix timestamp to a time.Time object.
	date = strings.ReplaceAll(date, "/Date(", "")
	date = strings.ReplaceAll(date, ")/", "")

	epoch, err := strconv.ParseInt(date, 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("corby: %s", err)
	}

	return time.Unix(epoch/1000, 0), nil
}

func (p *Provider) commonBinName(bin string) string {
	bin = strings.ToLower(strings.TrimSpace(bin))
	switch {
	case strings.Contains(bin, "refuse"), strings.Contains(bin, "authorised bin"):
		return "Black"
	case strings.Contains(bin, "recycling"):
		return "Blue"
	case strings.Contains(bin, "garden"):
		return "Green"
	case strings.Contains(bin, "food caddy"), strings.Contains(bin, "communal food bin"):
		return "Food"
	default:
		return fmt.Sprintf("Unknown (%s)", bin)
	}
}
