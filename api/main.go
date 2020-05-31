package main

import (
	"fmt"
	"log"
	"time"
	"bytes"
	"regexp"
	"strings"
	"context"
	"encoding/json"
	"encoding/base64"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/aws/aws-sdk-go/service/transcribeservice"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

type APIResponse struct {
	Message  string `json:"message"`
}

type Response events.APIGatewayProxyResponse

const layout       string = "2006-01-02 15:04"
const layout2      string = "20060102150405"
const languageCode string = "ja-JP"
const mediaFormat  string = "mp3"
const bucketName   string = "your-bucket-name"
const bucketRegion string = "ap-northeast-1"
const S3MediaPath  string = "media"

func HandleRequest(ctx context.Context, request events.APIGatewayProxyRequest) (Response, error) {
	var jsonBytes []byte
	var err error
	d := make(map[string]string)
	json.Unmarshal([]byte(request.Body), &d)
	if v, ok := d["action"]; ok {
		switch v {
		case "starttranscription" :
			if m, ok := d["mp3"]; ok {
				res, e := startTranscription(m)
				if e != nil {
					err = e
				} else {
					jsonBytes, _ = json.Marshal(APIResponse{Message: res})
				}
			}
		case "checkprogress" :
			if n, ok := d["name"]; ok {
				res, e := checkProgress(n)
				if e != nil {
					err = e
				} else {
					jsonBytes, _ = json.Marshal(APIResponse{Message: res})
				}
			}
		case "gettranscription" :
			if n, ok := d["name"]; ok {
				res, e := getTranscriptionJob(n)
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

func startTranscription(filedata string)(string, error) {
	t := time.Now()
	b64data := filedata[strings.IndexByte(filedata, ',')+1:]
	data, err := base64.StdEncoding.DecodeString(b64data)
	if err != nil {
		return "", err
	}
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(bucketRegion)},
	)
	if err != nil {
		return "", err
	}
	contentType := "audio/mp3"
	filename := t.Format(layout2) + ".mp3"
	uploader := s3manager.NewUploader(sess)
	_, err = uploader.Upload(&s3manager.UploadInput{
		ACL: aws.String("public-read"),
		Bucket: aws.String(bucketName),
		Key: aws.String(S3MediaPath + "/" + filename),
		Body: bytes.NewReader(data),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return "", err
	}
	url := "s3://" + bucketName + "/" + S3MediaPath + "/" + filename

	svc := transcribeservice.New(session.New(), &aws.Config{
		Region: aws.String("ap-northeast-1"),
	})

	input := &transcribeservice.StartTranscriptionJobInput{
		TranscriptionJobName: aws.String(filename),
		LanguageCode: aws.String(languageCode),
		OutputBucketName: aws.String(bucketName),
		MediaFormat: aws.String(mediaFormat),
		Media: &transcribeservice.Media{
			MediaFileUri: aws.String(url),
		},
	}
	_, err = svc.StartTranscriptionJob(input)
	if err != nil {
		return "", err
	}

	return filename, nil
}

func checkProgress(jobName string)(string, error) {
	svc := transcribeservice.New(session.New(), &aws.Config{
		Region: aws.String("ap-northeast-1"),
	})

	input := &transcribeservice.ListTranscriptionJobsInput{
		JobNameContains: aws.String(jobName),
	}
	res, err := svc.ListTranscriptionJobs(input)
	if err != nil {
		return "", err
	}
	return aws.StringValue(res.TranscriptionJobSummaries[0].TranscriptionJobStatus), nil
}

func getTranscriptionJob(jobName string)(string, error) {
	svc := transcribeservice.New(session.New(), &aws.Config{
		Region: aws.String("ap-northeast-1"),
	})

	input := &transcribeservice.GetTranscriptionJobInput{
		TranscriptionJobName: aws.String(jobName),
	}
	res, err := svc.GetTranscriptionJob(input)
	if err != nil {
		return "", err
	}
        url := aws.StringValue(res.TranscriptionJob.Transcript.TranscriptFileUri)
	rep := regexp.MustCompile(`\s*/\s*`)
	tmp := rep.Split(url, -1)
	svc_ := s3.New(session.New(), &aws.Config{
		Region: aws.String(bucketRegion),
	})
	obj, err2 := svc_.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(tmp[len(tmp) - 1]),
	})
	if err2 != nil {
		return "", err2
	}
	defer obj.Body.Close()

	buf := new(bytes.Buffer)
	buf.ReadFrom(obj.Body)
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

func main() {
	lambda.Start(HandleRequest)
}
