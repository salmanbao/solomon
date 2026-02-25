package memory

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"solomon/contexts/campaign-editorial/submission-service/domain/entities"
	domainerrors "solomon/contexts/campaign-editorial/submission-service/domain/errors"
	"solomon/contexts/campaign-editorial/submission-service/ports"

	"github.com/google/uuid"
)

type Store struct {
	mu sync.RWMutex

	submissions map[string]entities.Submission
	reports     map[string]entities.SubmissionReport
	flags       map[string]entities.SubmissionFlag
}

func NewStore(seed []entities.Submission) *Store {
	submissions := make(map[string]entities.Submission, len(seed))
	for _, item := range seed {
		submissions[item.SubmissionID] = item
	}
	return &Store{
		submissions: submissions,
		reports:     make(map[string]entities.SubmissionReport),
		flags:       make(map[string]entities.SubmissionFlag),
	}
}

func (s *Store) CreateSubmission(_ context.Context, submission entities.Submission) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, existing := range s.submissions {
		if existing.CampaignID == submission.CampaignID &&
			existing.CreatorID == submission.CreatorID &&
			existing.PostURL == submission.PostURL &&
			existing.Status != entities.SubmissionStatusCancelled {
			return domainerrors.ErrDuplicateSubmission
		}
	}
	s.submissions[submission.SubmissionID] = submission
	return nil
}

func (s *Store) UpdateSubmission(_ context.Context, submission entities.Submission) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.submissions[submission.SubmissionID]; !exists {
		return domainerrors.ErrSubmissionNotFound
	}
	s.submissions[submission.SubmissionID] = submission
	return nil
}

func (s *Store) GetSubmission(_ context.Context, submissionID string) (entities.Submission, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, exists := s.submissions[strings.TrimSpace(submissionID)]
	if !exists {
		return entities.Submission{}, domainerrors.ErrSubmissionNotFound
	}
	return item, nil
}

func (s *Store) ListSubmissions(_ context.Context, filter ports.SubmissionFilter) ([]entities.Submission, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]entities.Submission, 0, len(s.submissions))
	for _, item := range s.submissions {
		if strings.TrimSpace(filter.CreatorID) != "" && item.CreatorID != strings.TrimSpace(filter.CreatorID) {
			continue
		}
		if strings.TrimSpace(filter.CampaignID) != "" && item.CampaignID != strings.TrimSpace(filter.CampaignID) {
			continue
		}
		if filter.Status != "" && item.Status != filter.Status {
			continue
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
	return items, nil
}

func (s *Store) AddReport(_ context.Context, report entities.SubmissionReport) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.reports[report.ReportID] = report
	return nil
}

func (s *Store) AddFlag(_ context.Context, flag entities.SubmissionFlag) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.flags[flag.FlagID] = flag
	return nil
}

func (s *Store) Now() time.Time {
	return time.Now().UTC()
}

func (s *Store) NewID(_ context.Context) (string, error) {
	return uuid.NewString(), nil
}
