package main

import "log"

func handler(email string, ssn string) {
	log.Println("processing", email)
	redis.Set(ctx, "user", ssn)
}
