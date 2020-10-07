package main

import (
	"os"
	"fmt"
	"log"
	"time"
	"bytes"
	"regexp"
	"reflect"
	"strings"
	"context"
	"encoding/json"
	"encoding/base64"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/transcribe"
	ttypes "github.com/aws/aws-sdk-go-v2/service/transcribe/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	stypes "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type APIResponse struct {
	Message  string `json:"message"`
}

type Response events.APIGatewayProxyResponse

var cfg aws.Config
var s3Client *s3.Client
var transcribeClient *transcribe.Client

const layout      string = "2006-01-02 15:04"
const layout2     string = "20060102150405"
const S3MediaPath string = "media"

func HandleRequest(ctx context.Context, request events.APIGatewayProxyRequest) (Response, error) {
	var jsonBytes []byte
	var err error
	d := make(map[string]string)
	json.Unmarshal([]byte(request.Body), &d)
	if v, ok := d["action"]; ok {
		switch v {
		case "starttranscription" :
			if m, ok := d["mp3"]; ok {
				res, e := startTranscription(ctx, m)
				if e != nil {
					err = e
				} else {
					jsonBytes, _ = json.Marshal(APIResponse{Message: res})
				}
			}
		case "checkprogress" :
			if n, ok := d["name"]; ok {
				res, e := checkProgress(ctx, n)
				if e != nil {
					err = e
				} else {
					jsonBytes, _ = json.Marshal(APIResponse{Message: res})
				}
			}
		case "gettranscription" :
			if n, ok := d["name"]; ok {
				res, e := getTranscriptionJob(ctx, n)
				if e != nil {
					err = e
				} else {
					jsonBytes, _ = json.Marshal(APIResponse{Message: res})
				}
			}
		}
	}
	log.Print(request.RequestContext.Identity.SourceIP)
	if err != nil {
		log.Print(err)
		jsonBytes, _ = json.Marshal(APIResponse{Message: fmt.Sprint(err)})
		return Response{
			StatusCode: 500,
			Body: string(jsonBytes),
		}, nil
	}
	return Response {
		StatusCode: 200,
		Body: string(jsonBytes),
	}, nil
}

func startTranscription(ctx context.Context, filedata string)(string, error) {
	t := time.Now()
	b64data := filedata[strings.IndexByte(filedata, ',')+1:]
	data, err := base64.StdEncoding.DecodeString(b64data)
	if err != nil {
		return "", err
	}
	contentType := "audio/mp3"
	filename := t.Format(layout2) + ".mp3"
	if s3Client == nil {
		s3Client = getS3Client()
	}
	input := &s3.PutObjectInput{
		ACL: stypes.ObjectCannedACLPublicRead,
		Bucket: aws.String(os.Getenv("BUCKET_NAME")),
		Key: aws.String(S3MediaPath + "/" + filename),
		Body: bytes.NewReader(data),
		ContentType: aws.String(contentType),
	}
	_, err = s3Client.PutObject(ctx, input)
	if err != nil {
		return "", err
	}
	url := "s3://" + os.Getenv("BUCKET_NAME") + "/" + S3MediaPath + "/" + filename

	if transcribeClient == nil {
		transcribeClient = getTranscribeClient()
	}

	input_ := &transcribe.StartTranscriptionJobInput{
		TranscriptionJobName: aws.String(filename),
		LanguageCode: ttypes.LanguageCodeJa_jp,
		OutputBucketName: aws.String(os.Getenv("BUCKET_NAME")),
		MediaFormat: ttypes.MediaFormatMp3,
		Media: &ttypes.Media{
			MediaFileUri: aws.String(url),
		},
	}
	_, err = transcribeClient.StartTranscriptionJob(ctx, input_)
	if err != nil {
		return "", err
	}

	return filename, nil
}

func checkProgress(ctx context.Context, jobName string)(string, error) {
	if transcribeClient == nil {
		transcribeClient = getTranscribeClient()
	}

	input := &transcribe.ListTranscriptionJobsInput{
		JobNameContains: aws.String(jobName),
	}
	res, err := transcribeClient.ListTranscriptionJobs(ctx, input)
	if err != nil {
		return "", err
	}
	return string(res.TranscriptionJobSummaries[0].TranscriptionJobStatus), nil
}

func getTranscriptionJob(ctx context.Context, jobName string)(string, error) {
	if transcribeClient == nil {
		transcribeClient = getTranscribeClient()
	}

	input := &transcribe.GetTranscriptionJobInput{
		TranscriptionJobName: aws.String(jobName),
	}
	res, err := transcribeClient.GetTranscriptionJob(ctx, input)
	if err != nil {
		return "", err
	}
	url := stringValue(res.TranscriptionJob.Transcript.TranscriptFileUri)
	rep := regexp.MustCompile(`\s*/\s*`)
	tmp := rep.Split(url, -1)
	tmp_ := strings.Replace(tmp[len(tmp) - 1], "\"", "", 1)

	if s3Client == nil {
		s3Client = getS3Client()
	}
	input_ := &s3.GetObjectInput{
		Bucket: aws.String(os.Getenv("BUCKET_NAME")),
		Key:    aws.String(tmp_),
	}
	res2, err2 := s3Client.GetObject(ctx, input_)
	if err2 != nil {
		return "", err2
	}
	defer res2.Body.Close()

	buf := new(bytes.Buffer)
	buf.ReadFrom(res2.Body)
	res_ := buf.String()

	jsonBytes := ([]byte)(res_)
	var data interface{}

	if err3 := json.Unmarshal(jsonBytes, &data); err3 != nil {
		return "", err3
	}
	results := data.(map[string]interface{})["results"]
	results_, err4 := json.Marshal(results)
	if err4 != nil {
		return "", err4
	}

	return string(results_), nil
}

func getS3Client() *s3.Client {
	if cfg.Region != os.Getenv("REGION") {
		cfg = getConfig()
	}
	return s3.NewFromConfig(cfg)
}

func getTranscribeClient() *transcribe.Client {
	if cfg.Region != os.Getenv("REGION") {
		cfg = getConfig()
	}
	return transcribe.NewFromConfig(cfg)
}

func getConfig() aws.Config {
	var err error
	newConfig, err := config.LoadDefaultConfig()
	newConfig.Region = os.Getenv("REGION")
	if err != nil {
		log.Print(err)
	}
	return newConfig
}

func stringValue(i interface{}) string {
	var buf bytes.Buffer
	strVal(reflect.ValueOf(i), 0, &buf)
	return buf.String()
}

func strVal(v reflect.Value, indent int, buf *bytes.Buffer) {
	for v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	switch v.Kind() {
	case reflect.Struct:
		buf.WriteString("{\n")
		for i := 0; i < v.Type().NumField(); i++ {
			ft := v.Type().Field(i)
			fv := v.Field(i)
			if ft.Name[0:1] == strings.ToLower(ft.Name[0:1]) {
				continue // ignore unexported fields
			}
			if (fv.Kind() == reflect.Ptr || fv.Kind() == reflect.Slice) && fv.IsNil() {
				continue // ignore unset fields
			}
			buf.WriteString(strings.Repeat(" ", indent+2))
			buf.WriteString(ft.Name + ": ")
			if tag := ft.Tag.Get("sensitive"); tag == "true" {
				buf.WriteString("<sensitive>")
			} else {
				strVal(fv, indent+2, buf)
			}
			buf.WriteString(",\n")
		}
		buf.WriteString("\n" + strings.Repeat(" ", indent) + "}")
	case reflect.Slice:
		nl, id, id2 := "", "", ""
		if v.Len() > 3 {
			nl, id, id2 = "\n", strings.Repeat(" ", indent), strings.Repeat(" ", indent+2)
		}
		buf.WriteString("[" + nl)
		for i := 0; i < v.Len(); i++ {
			buf.WriteString(id2)
			strVal(v.Index(i), indent+2, buf)
			if i < v.Len()-1 {
				buf.WriteString("," + nl)
			}
		}
		buf.WriteString(nl + id + "]")
	case reflect.Map:
		buf.WriteString("{\n")
		for i, k := range v.MapKeys() {
			buf.WriteString(strings.Repeat(" ", indent+2))
			buf.WriteString(k.String() + ": ")
			strVal(v.MapIndex(k), indent+2, buf)
			if i < v.Len()-1 {
				buf.WriteString(",\n")
			}
		}
		buf.WriteString("\n" + strings.Repeat(" ", indent) + "}")
	default:
		format := "%v"
		switch v.Interface().(type) {
		case string:
			format = "%q"
		}
		fmt.Fprintf(buf, format, v.Interface())
	}
}

func main() {
	lambda.Start(HandleRequest)
}
