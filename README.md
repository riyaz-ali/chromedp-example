# Invoicify

This is an experimental repository containing a simple server that renders a dynamic invoice template 
(taken from [the awesome template by SparkSuite](https://github.com/sparksuite/simple-html-invoice-template))

The user can change the values in the form and submit and the server renders a PDF version of that using 
[`chromedp`](https://github.com/chromedp/chromedp) and [Headless Chrome](https://chromium.googlesource.com/chromium/src/+/master/headless/README.md)

## Running locally

1. Make sure you have a recent version of (stable) Chrome installed
2. Then run,
    ```shell
   > go run invoicify.go
   > # or go run invoicify.go --chrome <path_to_chrome> # if Chrome isn't installed at standard location
    ```