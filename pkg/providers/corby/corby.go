package corby

import (
	"context"
	"strings"
	"time"

	"github.com/Xiol/whatbin"
	"github.com/Xiol/whatbin/pkg/dateutils"
	"github.com/chromedp/chromedp"
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
	opts := append(chromedp.DefaultExecAllocatorOptions[:], chromedp.NoSandbox)
	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	var green string
	var blue string
	var black string
	var foodWaste string
	err := chromedp.Run(ctx,
		chromedp.Navigate("https://my.corby.gov.uk/service/Waste_Collection_Date"),
		chromedp.WaitVisible(`#address_search`),
		chromedp.SendKeys(`#address_search`, p.postcode),
		chromedp.Sleep(5*time.Second),
		chromedp.SendKeys(`#ChooseAddress`, p.firstLine),
		chromedp.Click(`#AF-Form-56765be8-8e4b-4a2d-9c9f-cfa55a71dab5 > div > nav > div.fillinButtonsRight > button`),
		chromedp.Sleep(5*time.Second),
		chromedp.WaitVisible(`#AF-Form-56d32560-ecaf-4763-87df-d544efa65a19 > section.all-sections.col-xs-12.AF-col-xs-fluid.col-sm-12 > section > div:nth-child(17) > div > span > table > thead > tr > th:nth-child(1)`),
		chromedp.Sleep(5*time.Second),
		chromedp.Text(`#WasteCollections > tr:nth-child(1) > td:nth-child(3) > h5`, &green, chromedp.NodeVisible),
		chromedp.Text(`#WasteCollections > tr:nth-child(2) > td:nth-child(3) > h5`, &blue, chromedp.NodeVisible),
		chromedp.Text(`#WasteCollections > tr:nth-child(3) > td:nth-child(3) > h5`, &black, chromedp.NodeVisible),
		chromedp.Text(`#WasteCollections > tr:nth-child(4) > td:nth-child(3) > h5`, &foodWaste, chromedp.NodeVisible),
	)
	if err != nil {
		return nil, err
	}

	var binsOut []string

	for k, v := range map[string]string{"Green": green, "Blue": blue, "Black": black, "Food Waste": foodWaste} {
		out, err := p.binOut(v)
		if err != nil {
			return nil, err
		}

		if out {
			binsOut = append(binsOut, k)
		}
	}

	if len(binsOut) == 0 {
		return nil, whatbin.ErrNoBinsToday
	}

	return binsOut, nil
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
