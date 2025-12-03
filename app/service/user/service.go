package user

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"sync/atomic"
	"time"

	errmsg "rankr/pkg/err_msg"
	"rankr/pkg/logger"
	"rankr/pkg/statuscode"
	types "rankr/type"
)

const (
	maxAllowedWorkers = 10
	defaultQueueSize  = 200
)

type Repository interface {
	UpsertUser(ctx context.Context, user User) error
	GetByID(ctx context.Context, id types.ID) (User, error)
}

type Validator interface {
	ValidateImportUser(u ImportUser) error
	ValidateUserID(id types.ID) error
}

type Service struct {
	repo       Repository
	validator  Validator
	maxWorkers int
	queueSize  int
}

func NewService(repo Repository, validator Validator, opts ImportOptions) Service {
	normalized := normalizeOptions(opts)
	return Service{
		repo:       repo,
		validator:  validator,
		maxWorkers: normalized.MaxWorkers,
		queueSize:  normalized.QueueSize,
	}
}

func normalizeOptions(opts ImportOptions) ImportOptions {
	res := opts
	if res.MaxWorkers <= 0 || res.MaxWorkers > maxAllowedWorkers {
		res.MaxWorkers = maxAllowedWorkers
	}
	if res.QueueSize <= 0 {
		res.QueueSize = defaultQueueSize
	}
	return res
}

func (s Service) ImportFromFile(ctx context.Context, filePath string) (ImportSummary, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return ImportSummary{}, fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	return s.Import(ctx, file)
}

func (s Service) Import(ctx context.Context, reader io.Reader) (ImportSummary, error) {
	start := time.Now()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	jobs := make(chan ImportUser, s.queueSize)
	errCh := make(chan error, s.maxWorkers*2)

	var (
		wg      sync.WaitGroup
		success int64
		log     = logger.L()
	)

	for i := 0; i < s.maxWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for raw := range jobs {
				if ctx.Err() != nil {
					return
				}

				if err := s.validator.ValidateImportUser(raw); err != nil {
					select {
					case errCh <- err:
					default:
					}
					continue
				}

				if err := s.repo.UpsertUser(ctx, raw.ToUser()); err != nil {
					select {
					case errCh <- err:
					default:
					}
					continue
				}
				atomic.AddInt64(&success, 1)
			}
		}()
	}

	total, readErr := streamUsers(ctx, reader, jobs)
	close(jobs)
	wg.Wait()
	close(errCh)

	summary := ImportSummary{
		Total:      total,
		Successful: int(success),
		Failed:     total - int(success),
		Duration:   time.Since(start),
	}

	var combined error
	var errCount int
	for err := range errCh {
		if err == nil {
			continue
		}
		errCount++
		if combined == nil {
			combined = err
		} else {
			combined = fmt.Errorf("%w; %v", combined, err)
		}
	}

	if readErr != nil {
		return summary, readErr
	}

	if errCount > 0 {
		log.Warn("import completed with errors", "failures", errCount)
	}

	return summary, combined
}

func (s Service) GetUser(ctx context.Context, id types.ID) (GetUserResponse, error) {
	if err := s.validator.ValidateUserID(id); err != nil {
		return GetUserResponse{}, err
	}

	res, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return GetUserResponse{}, errmsg.ErrorResponse{
				Message:         "user not found",
				Errors:          map[string]any{"user_id": id},
				InternalErrCode: statuscode.IntCodeNotFound,
			}
		}
		return GetUserResponse{}, errmsg.ErrorResponse{
			Message:         "failed to fetch user",
			Errors:          map[string]any{"user_id": id, "cause": err.Error()},
			InternalErrCode: statuscode.IntCodeUnExpected,
		}
	}
	if res.Addresses == nil {
		res.Addresses = []Address{}
	}
	return GetUserResponse{User: res}, nil
}
