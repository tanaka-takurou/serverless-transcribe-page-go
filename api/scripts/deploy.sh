#!/bin/bash
echo 'Updating API Lambda-Function...'
cd `dirname $0`/../
rm function.zip
rm bootstrap
GOARCH=arm64 GOOS=linux CGO_ENABLED=0 go build -o bootstrap main.go
zip -g function.zip bootstrap
aws lambda update-function-code \
	--profile default \
	--function-name your_api_function_name \
	--zip-file fileb://`pwd`/function.zip \
	--cli-connect-timeout 6000 \
	--publish
