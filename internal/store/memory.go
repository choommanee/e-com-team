package store

import (
	"context"
	"sort"
	"strings"
	"sync"

	"ecomteam/internal/domain"
)

// Memory is a thread-safe in-memory Store. It lets the whole app run with no
// database (used when DATABASE_URL is empty, e.g. local/mock mode).
type Memory struct {
	mu       sync.RWMutex
	users    map[string]domain.User         // id -> user
	byEmail  map[string]string              // email -> id
	subs     map[string]domain.Subscription // userID -> sub
	jobs     map[string]domain.Job          // id -> job
	usage    map[string]int                 // userID|period -> count
	affs     map[string]domain.Affiliate    // userID -> affiliate
	affCode  map[string]string              // code -> userID
	refs     map[string]domain.Referral     // referredUserID -> referral
}

// NewMemory returns an empty in-memory store.
func NewMemory() *Memory {
	return &Memory{
		users:   map[string]domain.User{},
		byEmail: map[string]string{},
		subs:    map[string]domain.Subscription{},
		jobs:    map[string]domain.Job{},
		usage:   map[string]int{},
		affs:    map[string]domain.Affiliate{},
		affCode: map[string]string{},
		refs:    map[string]domain.Referral{},
	}
}

func (m *Memory) Close() {}

func (m *Memory) CreateUser(_ context.Context, u domain.User) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := strings.ToLower(u.Email)
	if _, exists := m.byEmail[key]; exists {
		return ErrDuplicate
	}
	m.users[u.ID] = u
	m.byEmail[key] = u.ID
	return nil
}

func (m *Memory) GetUserByEmail(_ context.Context, email string) (domain.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	id, ok := m.byEmail[strings.ToLower(email)]
	if !ok {
		return domain.User{}, ErrNotFound
	}
	return m.users[id], nil
}

func (m *Memory) GetUserByID(_ context.Context, id string) (domain.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	u, ok := m.users[id]
	if !ok {
		return domain.User{}, ErrNotFound
	}
	return u, nil
}

func (m *Memory) GetSubscription(_ context.Context, userID string) (domain.Subscription, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.subs[userID]
	if !ok {
		return domain.Subscription{}, ErrNotFound
	}
	return s, nil
}

func (m *Memory) UpsertSubscription(_ context.Context, s domain.Subscription) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.subs[s.UserID] = s
	return nil
}

func (m *Memory) CreateJob(_ context.Context, j domain.Job) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.jobs[j.ID] = j
	return nil
}

func (m *Memory) GetJob(_ context.Context, id string) (domain.Job, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	j, ok := m.jobs[id]
	if !ok {
		return domain.Job{}, ErrNotFound
	}
	return j, nil
}

func (m *Memory) UpdateJob(_ context.Context, j domain.Job) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.jobs[j.ID]; !ok {
		return ErrNotFound
	}
	m.jobs[j.ID] = j
	return nil
}

func (m *Memory) ListJobsByUser(_ context.Context, userID string, limit, offset int) ([]domain.Job, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []domain.Job
	for _, j := range m.jobs {
		if j.UserID == userID {
			out = append(out, j)
		}
	}
	// Newest first.
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	if offset > len(out) {
		return nil, nil
	}
	out = out[offset:]
	if limit > 0 && limit < len(out) {
		out = out[:limit]
	}
	return out, nil
}

func (m *Memory) ListPendingJobs(_ context.Context) ([]domain.Job, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []domain.Job
	for _, j := range m.jobs {
		if j.Status == domain.JobPending || j.Status == domain.JobRunning {
			out = append(out, j)
		}
	}
	return out, nil
}

func (m *Memory) GetUsage(_ context.Context, userID, periodStart string) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.usage[userID+"|"+periodStart], nil
}

func (m *Memory) IncrementUsage(_ context.Context, userID, periodStart string) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := userID + "|" + periodStart
	m.usage[key]++
	return m.usage[key], nil
}

func (m *Memory) DecrementUsage(_ context.Context, userID, periodStart string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := userID + "|" + periodStart
	if m.usage[key] > 0 {
		m.usage[key]--
	}
	return nil
}

func (m *Memory) CreateAffiliate(_ context.Context, a domain.Affiliate) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.affs[a.UserID]; exists {
		return ErrDuplicate
	}
	if _, taken := m.affCode[a.Code]; taken {
		return ErrDuplicate
	}
	m.affs[a.UserID] = a
	m.affCode[a.Code] = a.UserID
	return nil
}

func (m *Memory) GetAffiliateByUser(_ context.Context, userID string) (domain.Affiliate, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	a, ok := m.affs[userID]
	if !ok {
		return domain.Affiliate{}, ErrNotFound
	}
	return a, nil
}

func (m *Memory) GetAffiliateByCode(_ context.Context, code string) (domain.Affiliate, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	uid, ok := m.affCode[code]
	if !ok {
		return domain.Affiliate{}, ErrNotFound
	}
	return m.affs[uid], nil
}

func (m *Memory) UpdateAffiliate(_ context.Context, a domain.Affiliate) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.affs[a.UserID]; !ok {
		return ErrNotFound
	}
	m.affs[a.UserID] = a
	m.affCode[a.Code] = a.UserID
	return nil
}

func (m *Memory) RecordReferral(_ context.Context, r domain.Referral) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.refs[r.ReferredUserID]; exists {
		return nil // first attribution wins; ignore duplicates
	}
	m.refs[r.ReferredUserID] = r
	if uid, ok := m.affCode[r.AffiliateCode]; ok {
		a := m.affs[uid]
		a.Signups++
		m.affs[uid] = a
	}
	return nil
}

func (m *Memory) GetReferralByUser(_ context.Context, referredUserID string) (domain.Referral, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	r, ok := m.refs[referredUserID]
	if !ok {
		return domain.Referral{}, ErrNotFound
	}
	return r, nil
}

func (m *Memory) MarkReferralConverted(_ context.Context, referredUserID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.refs[referredUserID]
	if !ok {
		return ErrNotFound
	}
	r.Converted = true
	m.refs[referredUserID] = r
	return nil
}
