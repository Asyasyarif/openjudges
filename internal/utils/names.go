package utils

import (
	"fmt"
	"math/rand"
	"time"
)

// GenerateJudgeName generates a default judge name in format "Judges-XXXX"
// where XXXX is a random 4-digit number
func GenerateJudgeName() string {
	rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("Judges-%04d", rand.Intn(10000))
}
