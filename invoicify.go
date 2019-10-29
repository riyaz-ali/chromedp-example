package main

import (
	"bytes"
	"context"
	"flag"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"github.com/gorilla/schema"
	"github.com/julienschmidt/httprouter"
	"github.com/leekchan/accounting"
	"html/template"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

var (
	tmpl, _ = template.New("").Funcs(
		template.FuncMap{
			"currency": func(amount float32) string {
				ac := &accounting.Accounting{Symbol: "₹", Precision: 2,}
				return ac.FormatMoney(amount)
			},
		},
	).ParseFiles("templates/index.html")

	decoder = schema.NewDecoder()
)

// Invoice describes the invoice model used when rendering templates and parsing form data
type Invoice struct {
	// The invoice number
	Number string

	// Invoice's creation date
	CreatedAt string

	// Invoice's due date
	DueAt string

	// Business information
	Business struct {
		// name of the business
		Name string

		// address lines
		Address string
	}

	// Customer's details
	Customer struct {
		// name of the customer
		Name string

		// address lines
		Address string

		// customer's contact (email / phone number)
		Contact string
	}

	// The items that are part of this invoice
	Items []struct {
		// item's name
		Name string

		// total price
		Price float32
	}
}

func (i *Invoice) Total() float32 {
	sum := float32(0)
	for _, p := range i.Items {
		sum += p.Price
	}
	return sum
}

func defaultInvoice() *Invoice {
	return &Invoice{
		Number:    "000",
		CreatedAt: time.Now().Format("Jan 01, 2006"),
		DueAt:     time.Now().Format("Jan 01, 2006"),
		Business: struct {
			Name    string
			Address string
		}{},
		Customer: struct {
			Name    string
			Address string
			Contact string
		}{},
		Items: []struct {
			Name  string
			Price float32
		}{
			{Name: "Apple MacBook Pro", Price: 1_00_000},
			{Name: "Apple Magic Mouse", Price: 10_000},
		},
	}
}

func main() {

	chrome := flag.String("chrome", "", "path to chrome binary; leave empty to use platform default")
	flag.Parse()

	log.Print("starting invoicify")

	// prepare the browser
	log.Print("preparing browser options")
	opts := chromedp.DefaultExecAllocatorOptions[:]
	if *chrome != "" {
		opts = append(opts, chromedp.ExecPath(*chrome))
	}

	log.Print("launching new browser window")
	browser, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	router := httprouter.New()

	// configure routes
	router.GET("/", index)
	router.POST("/", render(browser))

	router.PanicHandler = onPanic

	log.Print("listening for requests on port 8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}

// index renders the basic html invoice template for display in browser
// on the client side
func index(w http.ResponseWriter, r *http.Request, param httprouter.Params) {
	die(tmpl.ExecuteTemplate(w, "index.html", defaultInvoice()))
}

// render renders the incoming request in a headless chrome instance and
// generates and return a PDF file from that
func render(browser context.Context) func(w http.ResponseWriter, r *http.Request, param httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, param httprouter.Params) {
		// get the form values
		die(r.ParseForm())
		var invoice Invoice
		die(decoder.Decode(&invoice, r.PostForm))
		invoice.Items = []struct {
			Name  string
			Price float32
		}{
			{Name: "Apple MacBook Pro", Price: 1_00_000},
			{Name: "Apple Magic Mouse", Price: 10_000},
		}

		// render the template
		var buffer strings.Builder
		err := tmpl.ExecuteTemplate(&buffer, "index.html", &invoice)
		die(err)

		// open the rendered file in chrome and take a snapshot
		tab, cancel := chromedp.NewContext(browser)
		defer cancel()

		err = chromedp.Run(tab,
			chromedp.Navigate("about:blank"), // open a blank page
			chromedp.ActionFunc( // render the template
				func(c context.Context) error {
					ctx := chromedp.FromContext(c)
					return page.SetDocumentContent(cdp.FrameID(ctx.Target.TargetID.String()), buffer.String()).Do(c)
				}),
			chromedp.ActionFunc( // print to pdf
				func(c context.Context) error {
					pdf := page.PrintToPDF()
					pdf.PrintBackground = true

					data, _, err := pdf.Do(c)
					if err != nil {
						return err
					}
					// write data as http response
					w.Header().Set("content-disposition", "inline")
					w.Header().Set("content-type", "application/pdf")
					_, err = io.Copy(w, bytes.NewBuffer(data))
					return err
				}),
		)
		die(err)
	}
}

// onPanic is used by httprouter to recover from panic(..)
// This handler should generate an appropriate error response and
// send it to the client
func onPanic(w http.ResponseWriter, _ *http.Request, _ interface{}) {
	http.Error(w, "Server Error", 500)
}

// die prints an error to the error stream and panics
func die(err error) {
	if err != nil {
		log.Printf("[ERROR]: %v", err)
		panic(err)
	}
}
