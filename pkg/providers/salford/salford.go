package salford

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Xiol/whatbin"
	"github.com/Xiol/whatbin/pkg/dateutils"
	"github.com/gocolly/colly/v2"
	log "github.com/sirupsen/logrus"
)

const dateLayout = "Monday _2 January 2006"

type Provider struct {
	houseNum int
	postcode string
	client   whatbin.Doer
}

type Bins struct {
	Pink  []time.Time
	Blue  []time.Time
	Brown []time.Time
	Black []time.Time
}

func New(houseNum int, postcode string, doer whatbin.Doer) *Provider {
	return &Provider{
		houseNum: houseNum,
		postcode: postcode,
		client:   doer,
	}
}

func (p *Provider) Bins() ([]string, error) {
	var bins []string

	cols, err := p.getNextCollections()
	if err != nil {
		return nil, err
	}

	tomorrow := time.Now().AddDate(0, 0, 1)

	if dateutils.Contains(cols.Black, tomorrow) {
		bins = append(bins, "Black")
	}
	if dateutils.Contains(cols.Brown, tomorrow) {
		bins = append(bins, "Brown")
	}
	if dateutils.Contains(cols.Blue, tomorrow) {
		bins = append(bins, "Blue")
	}
	if dateutils.Contains(cols.Pink, tomorrow) {
		bins = append(bins, "Pink")
	}

	if len(bins) == 0 {
		return nil, whatbin.ErrNoBinsToday
	}
	return bins, nil
}

func (p *Provider) getNextCollections() (Bins, error) {
	bins := Bins{}
	uprnHref, err := p.getUPRNLink()
	if err != nil {
		return bins, err
	}

	c := colly.NewCollector(
		colly.AllowedDomains("www.salford.gov.uk"),
	)

	c.OnHTML("div[class=clearfix]", func(h *colly.HTMLElement) {
		h.ForEach("div div", func(i int, h *colly.HTMLElement) {
			var err error
			switch h.Attr("class") {
			case "black":
				bins.Black, err = p.extractDates(h)
			case "pink":
				bins.Pink, err = p.extractDates(h)
			case "bluebrown":
				if h.ChildText("strong") == "Blue bins:" {
					bins.Blue, err = p.extractDates(h)
				}
				if h.ChildText("strong") == "Brown bins:" {
					bins.Brown, err = p.extractDates(h)
				}
			}
			if err != nil {
				log.WithError(err).Error("salford.Provider: date extraction issue")
				return
			}
		})
	})

	err = c.Visit("https://www.salford.gov.uk" + uprnHref)
	if err != nil {
		return bins, err
	}

	return bins, nil
}

func (p *Provider) extractDates(h *colly.HTMLElement) ([]time.Time, error) {
	var innerErr error
	var dates []time.Time
	h.ForEachWithBreak("ul li", func(i int, h *colly.HTMLElement) bool {
		ptime, err := time.Parse(dateLayout, h.Text)
		if err != nil {
			innerErr = err
			return false
		}
		dates = append(dates, ptime)
		return true
	})

	if innerErr != nil {
		return nil, innerErr
	}
	return dates, nil
}

func (p *Provider) getUPRNLink() (string, error) {
	var href string

	c := colly.NewCollector(
		colly.AllowedDomains("www.salford.gov.uk"),
	)

	payload := url.Values{
		"prop": []string{strings.ToUpper(p.postcode)},
	}

	c.OnHTML("ul[class=properties]", func(h *colly.HTMLElement) {
		h.ForEachWithBreak("li", func(i int, h *colly.HTMLElement) bool {
			if strings.HasPrefix(h.ChildText("a"), strconv.Itoa(p.houseNum)+" ") {
				href = h.ChildAttr("a", "href")
				log.WithFields(log.Fields{
					"href":     href,
					"houseNum": p.houseNum,
					"postcode": p.postcode,
				}).Debug("found UPRN link for property")
				return false
			}
			return true
		})
	})

	err := c.Request("POST",
		"https://www.salford.gov.uk/bins-and-recycling/bin-collection-days",
		strings.NewReader(payload.Encode()),
		nil,
		whatbin.ImpersonationHeaders,
	)
	if err != nil {
		return "", fmt.Errorf("salford.Provider: failed to make POST request with postcode: %s", err)
	}

	if href != "" {
		return href, nil
	}
	return "", fmt.Errorf("salford.Provider: could not find house %d at %s", p.houseNum, p.postcode)
}
