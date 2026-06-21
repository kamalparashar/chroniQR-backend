package services

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// alphabet excludes ambiguous chars: 0/O, 1/l/I
const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghjkmnpqrstuvwxyz23456789"

var randomGenerator = rand.New(rand.NewSource(time.Now().UnixNano()))

func codeLength() int {
	n, err := strconv.Atoi(os.Getenv("SHORT_CODE_LENGTH"))
	if err != nil || n <= 0 {
		return 8
	}
	return n
}

func nanoid(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = alphabet[randomGenerator.Intn(len(alphabet))]
	}
	return string(b)
}

// GenerateShortCode generates a unique short code, retrying up to 5 times.
func GenerateShortCode(ctx context.Context, db *pgxpool.Pool) (string, error) {
	length := codeLength()
	for attempt := 0; attempt < 5; attempt++ {
		code := nanoid(length)
		var exists bool
		err := db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM qr_codes WHERE short_code = $1)", code).Scan(&exists)
		if err != nil {
			return "", err
		}
		if !exists {
			return code, nil
		}
	}
	return "", fmt.Errorf("short code generation exhausted retries — try again")
}
