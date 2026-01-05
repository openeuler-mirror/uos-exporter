package ipmi

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

type Client struct {
	Host     string
	User     string
	Password string
	Timeout  time.Duration
	Retries  int
}


// TODO: implement functions
