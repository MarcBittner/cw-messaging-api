#!/usr/bin/env bash

  # resove program dependecie as a creature comfort for the end user

go get github.com/brpaz/echozap
go get github.com/labstack/echo/v4
go get github.com/labstack/echo/v4/middleware
go get github.com/sendgrid/sendgrid-go
go get github.com/sendgrid/sendgrid-go/helpers/mail
go get go.uber.org/zap

  # Read the key-value pairs from the file

while read -r line; do

  # Split the line into key and value

  key=$(echo "$line" | cut -d '=' -f 1)
  value=$(echo "$line" | cut -d '=' -f 2)

  # Export the key-value pair as an environment variable

  export "$key=$value"
done < "$1"

# Execute the command

exec "$@"
