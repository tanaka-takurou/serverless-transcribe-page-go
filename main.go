package main

import (
	"io"
	"os"
	"log"
	"bytes"
	"embed"
	"context"
	"html/template"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

type PageData struct {
	Title   string
	ApiPath string
}

type Response events.APIGatewayProxyResponse

//go:embed templates
var templateFS embed.FS

const title string = "Sample Transcribe Page"

func HandleRequest(ctx context.Context, request events.APIGatewayProxyRequest) (Response, error) {
	tmp := template.New("tmp")
	var dat PageData
	funcMap := template.FuncMap{
		"safehtml": func(text string) template.HTML { return template.HTML(text) },
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
		"mul": func(a, b int) int { return a * b },
		"div": func(a, b int) int { return a / b },
	}
	buf := new(bytes.Buffer)
	fw := io.Writer(buf)
	dat.Title = title
	dat.ApiPath = os.Getenv("API_PATH")
	tmp = template.Must(template.New("").Funcs(funcMap).ParseFS(templateFS, "templates/index.html", "templates/view.html", "templates/header.html"))
	if e := tmp.ExecuteTemplate(fw, "base", dat); e != nil {
		log.Fatal(e)
	}
	res := Response{
		StatusCode:      200,
		IsBase64Encoded: false,
		Body:            string(buf.Bytes()),
		Headers: map[string]string{
			"Content-Type": "text/html",
		},
	}
	return res, nil
}

func main() {
	lambda.Start(HandleRequest)
}
