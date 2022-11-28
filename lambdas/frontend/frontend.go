package main

import (
	"context"
	"strings"
	"text/template"

	"github.com/apex/log"
	"github.com/aquasecurity/lmdrouter"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"gjhr.me/newsletter/data/list"
	"gjhr.me/newsletter/data/subscription"
	"gjhr.me/newsletter/providers/config"
	"gjhr.me/newsletter/subscriptionflow"
	"gjhr.me/newsletter/utils/loggermiddleware"
)

var router *lmdrouter.Router
var templates *template.Template

type htmlContent struct {
	Title        string
	List         *list.List
	Subscription *subscription.Subscription
	Err          error
}

func init() {
	envLogLevel := config.Get().LogLevel
	if envLogLevel != "" {
		log.SetLevelFromString(envLogLevel)
	}

	var err error
	router = lmdrouter.NewRouter("", loggermiddleware.LoggerMiddleware)

	// GETs because these are primarily interacted through with links
	router.Route("GET", "/", root)
	router.Route("GET", "/subscribe", subscribe)
	router.Route("GET", "/verify", verify)
	router.Route("GET", "/unsubscribe", unsubscribe)

	// Templates
	templates = template.New("root")

	_, err = templates.New("head").Parse(`
	<!doctype html>
	<html lang=en>
	<head>
	<meta charset=utf-8>
	<meta http-equiv=x-ua-compatible content="IE=edge">
	<meta name=viewport content="width=device-width,initial-scale=1">
	<title>{{.Title}}</title>
	<style type=text/css>body{margin:auto;max-width:650px;line-height:1.6;font-size:18px;color:#444;padding:0 10px}h1,h2,h3{line-height:1.2}a,a:visited{color:#333;text-decoration-color:#19c7e5;text-decoration-thickness:2px}footer{margin-top:10px}time{font-style:italic}figure{margin:0}figcaption{text-align:center;font-size:.7em}hr{width:80%;border:1px solid #d3d3d3}.flex-spaced{display:flex;flex-wrap:wrap;align-items:center;justify-content:space-around}.svg-icon{width:16px;height:16px;fill:#444}.svg-inline{display:none}#main-header div{flex-grow:100}#main-header div a{margin-left:5px}#main-footer div{margin-left:5px}#about-short{display:flex;flex-wrap:wrap;align-items:center;justify-content:center;margin:20px 0;padding:10px;gap:10px;border-radius:10px;border:1px solid #d3d3d3}#about-short img{border-radius:50%}#about-short div{flex-grow:1;width:75%;min-width:75%;max-width:100%}#about-short form{width:100%;display:flex;justify-content:center}input{border:1px solid #d3d3d3;padding:10px;margin:5px;border-radius:5px}.about-short-submit{color:#fff;background-color:#0f9afc}img{max-width:100%}pre{white-space:pre-wrap;font-size:.75em}.small{font-size:.7em}.index-list li{padding-bottom:.7em}ul{list-style-type:none;padding:0;line-height:1.2}ul li:not(:last-child){margin-bottom:.5em}</style>
	</head>
	
	<body>
		<h1>{{.Title}}</h1>
		<div id="content">`)
	if err != nil {
		panic(err)
	}

	_, err = templates.New("foot").Parse(`
		</div>
	</body>
	</html>`)
	if err != nil {
		panic(err)
	}

	_, err = templates.New("index").Parse(`
	{{ template "head" . }}
	<form method="get" action="/subscribe">
		<input type="text" id="email" name="email">
		<input type="submit" value="Subscribe">
	</form>
	<p>
		{{.List.Description}}
	</p>
	<details>
		<summary>Unsubscribe</summary>
		<form method="get" action="/unsubscribe">
			<input type="text" id="email" name="email">
			<input type="submit" value="Unsubscribe">
		</form>
	</details>
	{{ template "foot" . }}
	`)
	if err != nil {
		panic(err)
	}

	_, err = templates.New("subscribe").Parse(`
	{{ template "head" . }}
	Please follow the verification link sent to '{{ .Subscription.Email }}' to confirm your subscription.
	{{ template "foot" . }}
	`)
	if err != nil {
		panic(err)
	}

	_, err = templates.New("unsubscribe").Parse(`
	{{ template "head" . }}
	You have unsubscribed from '{{ .List.Name }}'.
	{{ template "foot" . }}
	`)
	if err != nil {
		panic(err)
	}

	_, err = templates.New("verify").Parse(`
	{{ template "head" . }}
	Subscription verified!
	{{ template "foot" . }}
	`)
	if err != nil {
		panic(err)
	}

	_, err = templates.New("error").Parse(`
	{{ template "head" . }}
	Unexpected error has occured: {{.Err}}
	{{ template "foot" . }}
	`)
	if err != nil {
		panic(err)
	}
}

func main() {
	InternalMain()
}

func InternalMain() {
	lambda.Start(router.Handler)
}

func root(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	list, err := list.GetFromDomain(req.RequestContext.DomainName)
	if err != nil {
		return returnErr(err)
	}
	return returnHtml(200, "index", htmlContent{Title: list.Name, List: list})
}

func subscribe(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	list, err := list.GetFromDomain(req.RequestContext.DomainName)
	if err != nil {
		return returnErr(err)
	}
	subscription, err := listmanagement.Subscribe(list, req.QueryStringParameters["email"])
	if err != nil {
		return returnErr(err)
	}
	return returnHtml(200, "subscribe", htmlContent{Title: "Verification Needed", List: list, Subscription: subscription})
}

func unsubscribe(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	list, err := list.GetFromDomain(req.RequestContext.DomainName)
	if err != nil {
		return returnErr(err)
	}
	err = listmanagement.Unsubscribe(list.Name, req.QueryStringParameters["email"])
	if err != nil {
		return returnErr(err)
	}
	return returnHtml(200, "unsubscribe", htmlContent{Title: "Unsubscribed", List: list})
}

func verify(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	list, err := list.GetFromDomain(req.RequestContext.DomainName)
	if err != nil {
		return returnErr(err)
	}
	err = listmanagement.Verify(req.QueryStringParameters["token"])
	if err != nil {
		return returnErr(err)
	}
	return returnHtml(200, "verify", htmlContent{Title: "Welcome", List: list})
}

// todo make errors HTTPErrors and handle automatically
func returnErr(err error) (events.APIGatewayProxyResponse, error) {
	log.Errorf("Unexpected uncaught error: %v", err)
	return returnHtml(500, "error", htmlContent{
		Title: "Error",
		Err:   err,
	})
}

func returnHtml(status int, templateName string, content htmlContent) (events.APIGatewayProxyResponse, error) {
	b := &strings.Builder{}
	err := templates.ExecuteTemplate(b, templateName, content)
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
