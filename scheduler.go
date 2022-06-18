package main

import (
	"github.com/go-co-op/gocron"
	"time"
)

var Scheduler = gocron.NewScheduler(time.UTC)
