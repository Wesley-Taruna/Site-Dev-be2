package repository

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"strconv"
	"time"

	"site-dev-backend/global"

	"github.com/go-redis/redis/v8"
	"gitlab.com/fami-app-v2/familib/familabs/opt"
)

// ErrApprovalCodeNotFound: the code is not in Redis (never existed, expired
// or mistyped).
var ErrApprovalCodeNotFound = errors.New("idgen: approval code not found or expired")

const (
	approvalCodeCharset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	approvalCodeLength  = 32
	// ApprovalCodeTTL is how long an e-mail link code works after its
	// level became PENDING.
	ApprovalCodeTTL = 24 * time.Hour
)

// GenerateSecureApprovalCode returns 32 random letters/digits
// (ApprovalDET.SecureApprovalCode), same as site-dev-api.
func GenerateSecureApprovalCode() (string, error) {
	raw := make([]byte, approvalCodeLength)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("idgen: failed to generate approval code: %w", err)
	}
	for i, b := range raw {
		raw[i] = approvalCodeCharset[int(b)%len(approvalCodeCharset)]
	}
	return string(raw), nil
}

// IApprovalCodeRepository maps an e-mail link code to its ApprovalDET row in
// Redis (key "<app>:<env>:approval_code:<code>:").
type IApprovalCodeRepository interface {
	Set(ctx context.Context, code string, detailID uint) (err error)
	Get(ctx context.Context, code string) (detailID uint, err error)
}

type ApprovalCodeRepository struct {
	opt opt.Options
}

func NewApprovalCodeRepository() IApprovalCodeRepository {
	return &ApprovalCodeRepository{
		opt: global.Setup.Common,
	}
}

func (r *ApprovalCodeRepository) key(code string) string {
	return redisKey(r.opt, fmt.Sprintf("approval_code:%s", code))
}

func (r *ApprovalCodeRepository) Set(ctx context.Context, code string, detailID uint) (err error) {
	ctx, span := r.opt.Trace.Start(ctx, "ApprovalCodeRepository.Set")
	defer span.End()

	client, err := redisClient(r.opt, "idgen")
	if err != nil {
		span.RecordError(err)
		return
	}

	if err = client.Set(ctx, r.key(code), detailID, ApprovalCodeTTL).Err(); err != nil {
		span.RecordError(err)
		return
	}

	return
}

func (r *ApprovalCodeRepository) Get(ctx context.Context, code string) (detailID uint, err error) {
	ctx, span := r.opt.Trace.Start(ctx, "ApprovalCodeRepository.Get")
	defer span.End()

	client, err := redisClient(r.opt, "idgen")
	if err != nil {
		span.RecordError(err)
		return
	}

	val, err := client.Get(ctx, r.key(code)).Result()
	if errors.Is(err, redis.Nil) {
		err = ErrApprovalCodeNotFound
		span.RecordError(err)
		return
	}
	if err != nil {
		err = fmt.Errorf("idgen: failed to read approval code from redis: %w", err)
		span.RecordError(err)
		return
	}

	id, err := strconv.ParseUint(val, 10, 64)
	if err != nil {
		err = fmt.Errorf("idgen: corrupt approval code value in redis: %w", err)
		span.RecordError(err)
		return
	}
	detailID = uint(id)

	return
}
