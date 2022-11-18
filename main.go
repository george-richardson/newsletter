package main

import (
	"context"
	"fmt"
	"strings"
	"text/template"

	"github.com/aquasecurity/lmdrouter"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"gjhr.me/newsletter/listmanagement"
)

var router *lmdrouter.Router
var htmlTemplate *template.Template

type htmlContent struct {
	Title   string
	Content string
}

func init() {
	var err error
	router = lmdrouter.NewRouter("")

	// GETs because these are primarily interacted through with links
	router.Route("GET", "/", root)
	router.Route("GET", "/subscribe", subscribe)
	router.Route("GET", "/verify", verify)
	router.Route("GET", "/unsubscribe", unsubscribe)

	// Template
	htmlTemplate, err = template.New("default").Parse(`
	<!doctype html>
	<html lang=en>
	<head>
	<meta charset=utf-8>
	<meta http-equiv=x-ua-compatible content="IE=edge">
	<meta name=viewport content="width=device-width,initial-scale=1">
	<title>Newsletter</title>
	<style type=text/css>body{margin:auto;max-width:650px;line-height:1.6;font-size:18px;color:#444;padding:0 10px}h1,h2,h3{line-height:1.2}a,a:visited{color:#333;text-decoration-color:#19c7e5;text-decoration-thickness:2px}footer{margin-top:10px}time{font-style:italic}figure{margin:0}figcaption{text-align:center;font-size:.7em}hr{width:80%;border:1px solid #d3d3d3}.flex-spaced{display:flex;flex-wrap:wrap;align-items:center;justify-content:space-around}.svg-icon{width:16px;height:16px;fill:#444}.svg-inline{display:none}#main-header div{flex-grow:100}#main-header div a{margin-left:5px}#main-footer div{margin-left:5px}#about-short{display:flex;flex-wrap:wrap;align-items:center;justify-content:center;margin:20px 0;padding:10px;gap:10px;border-radius:10px;border:1px solid #d3d3d3}#about-short img{border-radius:50%}#about-short div{flex-grow:1;width:75%;min-width:75%;max-width:100%}#about-short form{width:100%;display:flex;justify-content:center}input{border:1px solid #d3d3d3;padding:10px;margin:5px;border-radius:5px}.about-short-submit{color:#fff;background-color:#0f9afc}img{max-width:100%}pre{white-space:pre-wrap;font-size:.75em}.small{font-size:.7em}.index-list li{padding-bottom:.7em}ul{list-style-type:none;padding:0;line-height:1.2}ul li:not(:last-child){margin-bottom:.5em}</style>
	</head>
	
	<body>
		<h1>{{.Title}}</h1>
		<div id="content">
			{{.Content}}
		</div>
	</body>
	</html>`)
	if err != nil {
		panic(err)
	}
}

func main() {
	lambda.Start(router.Handler)
}

func root(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	//todo stop hardcoding list name
	//todo stop hardcoding action path
	return returnHtml(200, htmlContent{"Newsletter", `
	<form method="get" action="/prod/subscribe">
		<input type="text" id="email" name="email">
		<input type="hidden" id="list" name="list" value="gjhr.me">
		<input type="submit" value="Subscribe">
	</form>
	<form method="get" action="/prod/unsubscribe">
		<input type="text" id="email" name="email">
		<input type="hidden" id="list" name="list" value="gjhr.me">
		<input type="submit" value="Unsubscribe">
	</form>
	`})
}

func subscribe(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	list, email := req.QueryStringParameters["list"], req.QueryStringParameters["email"]
	err := listmanagement.Subscribe(list, email)
	if err != nil {
		return returnErr(err)
	}
	return returnHtml(200, htmlContent{"Verification required!", fmt.Sprintf("We have sent an email to '%v' for verification.", email)})
}

func unsubscribe(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	list, email := req.QueryStringParameters["list"], req.QueryStringParameters["email"]
	err := listmanagement.Unsubscribe(list, email)
	if err != nil {
		return returnErr(err)
	}
	return returnHtml(200, htmlContent{"Unsubscribed!", fmt.Sprintf("You are now unsubscribed from %v.", list)})
}

func verify(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	err := listmanagement.Verify(req.QueryStringParameters["token"])
	if err != nil {
		return returnErr(err)
	}
	return returnHtml(200, htmlContent{"Email Verified!", "Thank you for verifying your subscription."})
}

// todo make errors HTTPErrors and handle automatically
func returnErr(err error) (events.APIGatewayProxyResponse, error) {
	return returnHtml(500, htmlContent{Title: "Error", Content: fmt.Sprintf("Unexpected error has occurred: %v", err)})
}

func returnHtml(status int, content htmlContent) (events.APIGatewayProxyResponse, error) {
	b := &strings.Builder{}
	err := htmlTemplate.Execute(b, content)
	if err != nil {
		return lmdrouter.HandleError(err) // Fallback error handler
	}
	response := events.APIGatewayProxyResponse{
		StatusCode: status,
		Headers: map[string]string{
			"Content-Type": "text/html",
		},
		Body: b.String(),
	}

	return response, nil
}
