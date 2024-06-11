package env

import "os"

var IsDevelopment = os.Getenv("ENVIRONMENT") == "dev"
