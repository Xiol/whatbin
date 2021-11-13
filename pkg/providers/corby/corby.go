package corby

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Xiol/whatbin"
	"github.com/Xiol/whatbin/pkg/dateutils"
	"github.com/chromedp/chromedp"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type BinColour string

const (
	Green     BinColour = "Green"
	Blue      BinColour = "Blue"
	Black     BinColour = "Black"
	FoodWaste BinColour = "Food Waste"
)

type Provider struct {
	firstLine string
	postcode  string
}

func New(firstLine, postcode string) *Provider {
	return &Provider{
		firstLine: firstLine,
		postcode:  postcode,
	}
}

func (p *Provider) Bins() ([]string, error) {
	log.Info("corby: initialising provider")

	opts := append(chromedp.DefaultExecAllocatorOptions[:], chromedp.NoSandbox)
	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Infof))
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 300*time.Second)
	defer cancel()

	log.Info("corby: running scrape for https://my.corby.gov.uk/service/Waste_Collection_Date")

	var type1, date1, type2, date2, type3, date3, type4, date4 string
	err := chromedp.Run(ctx,
		chromedp.Navigate("https://my.corby.gov.uk/service/Waste_Collection_Date"),
		chromedp.WaitVisible(`#address_search`),
		chromedp.SendKeys(`#address_search`, p.postcode),
		chromedp.Sleep(5*time.Second),
		chromedp.SendKeys(`#ChooseAddress`, p.firstLine),
		chromedp.Click(`#AF-Form-56765be8-8e4b-4a2d-9c9f-cfa55a71dab5 > div > nav > div.fillinButtonsRight > button`),
		chromedp.Sleep(5*time.Second),
		chromedp.WaitVisible(`#AF-Form-56d32560-ecaf-4763-87df-d544efa65a19 > section.all-sections.col-xs-12.AF-col-xs-fluid.col-sm-12 > section > div:nth-child(17) > div > span > table > thead > tr > th:nth-child(1)`),
		chromedp.Sleep(20*time.Second),
		chromedp.Text(`#WasteCollections > tr:nth-child(1) > td:nth-child(3) > h5`, &date1, chromedp.NodeVisible),
		chromedp.Text(`#WasteCollections > tr:nth-child(2) > td:nth-child(3) > h5`, &date2, chromedp.NodeVisible),
		chromedp.Text(`#WasteCollections > tr:nth-child(3) > td:nth-child(3) > h5`, &date3, chromedp.NodeVisible),
		chromedp.Text(`#WasteCollections > tr:nth-child(4) > td:nth-child(3) > h5`, &date4, chromedp.NodeVisible),
		chromedp.Text(`#WasteCollections > tr:nth-child(1) > td:nth-child(2) > b`, &type1, chromedp.NodeVisible),
		chromedp.Text(`#WasteCollections > tr:nth-child(2) > td:nth-child(2) > b`, &type2, chromedp.NodeVisible),
		chromedp.Text(`#WasteCollections > tr:nth-child(3) > td:nth-child(2) > b`, &type3, chromedp.NodeVisible),
		chromedp.Text(`#WasteCollections > tr:nth-child(4) > td:nth-child(2) > b`, &type4, chromedp.NodeVisible),
	)
	if err != nil {
		return nil, err
	}

	log.WithFields(log.Fields{
		type1: date1,
		type2: date2,
		type3: date3,
		type4: date4,
	}).Info("corby: retrieved dates for bins")

	var binsOut []string

	for _, td := range [][]string{{type1, date1}, {type2, date2}, {type3, date3}, {type4, date4}} {
		out, err := p.binOut(td[1])
		if err != nil {
			return nil, err
		}

		if out {
			id, err := p.identify(td[0])
			if err != nil {
				return nil, err
			}

			binsOut = append(binsOut, string(id))
			if id == Blue && viper.GetBool("corby_green_out_with_blue") {
				binsOut = append(binsOut, string(Green))
			}
		}
	}

	if len(binsOut) == 0 {
		log.Info("corby: no bins out today")
		return nil, whatbin.ErrNoBinsToday
	}

	log.WithField("bins", binsOut).Info("corby: bins out today")

	return binsOut, nil
}

func (p *Provider) identify(t string) (BinColour, error) {
	if strings.Index(t, "Green Garden") == 0 {
		return Green, nil
	}

	if strings.Index(t, "Brown or Blue") == 0 {
		return Blue, nil
	}

	if strings.Index(t, "Green Food") == 0 {
		return FoodWaste, nil
	}

	if strings.Index(t, "Black") == 0 {
		return Black, nil
	}

	return BinColour(""), fmt.Errorf("corby: unable to identify bin colour for '%s'", t)
}

func (p *Provider) binOut(d string) (bool, error) {
	if d == "Tomorrow" {
		return true, nil
	}

	if strings.Contains(d, "null") || d == "Today" {
		return false, nil
	}

	t, err := time.Parse("Friday, 02 January 2006", d)
	if err != nil {
		return false, err
	}

	if dateutils.OutTomorrow(t) {
		return true, nil
	}
	return false, nil
}
