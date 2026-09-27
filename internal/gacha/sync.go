package gacha

import (
	"context"
	"crypto/rand"
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

var ErrSync = errors.New("gacha sync expired or invalid")

type RemotePage struct {
	Records  []Record `json:"list"`
	NextID   string   `json:"next_end_id"`
	More     bool     `json:"has_more"`
	Timezone int      `json:"timezone"`
	Language string   `json:"lang"`
}
type FetchPage func(context.Context, string, string, int) (RemotePage, error)
type SyncInfo struct {
	Ref      string        `json:"ref"`
	State    string        `json:"state"`
	Sequence int           `json:"sequence"`
	Pool     string        `json:"pool"`
	Pages    int           `json:"pages"`
	Fetched  int           `json:"fetched"`
	Result   *ImportResult `json:"result,omitempty"`
}

// SyncChoice is the account role a sync reads, or Link for a gacha link
// pasted in chat, whose authkey the caller keeps.
type SyncChoice struct {
	AccountRef, RoleRef string
	Link                bool
}

// syncJob is one sync. mu is held while a step reads a page, so whatever
// holds Syncs.mu never waits for it: the expiry Start checks is atomic, and
// Forget and Reset let go of Syncs.mu before they touch a job.
type syncJob struct {
	mu sync.Mutex
	// expires is when the sync lapses, in Unix nanoseconds.
	expires atomic.Int64
	view    SyncInfo
	choice  SyncChoice
	archive Archive
	version uint64
	full    bool
	known   map[string]Record
	pools   []string
	pool    int
	page    int
	endID   string
}

// expired reports whether the sync lapsed by now.
func (j *syncJob) expired(now time.Time) bool { return now.UnixNano() > j.expires.Load() }

// extend keeps the sync for another fifteen minutes from now.
func (j *syncJob) extend(now time.Time) { j.expires.Store(now.Add(15 * time.Minute).UnixNano()) }

type Syncs struct {
	mu   sync.Mutex
	jobs map[string]*syncJob
}

func (s *Syncs) Reset() {
	s.mu.Lock()
	jobs := s.jobs
	s.jobs = nil
	s.mu.Unlock()
	for _, j := range jobs {
		j.drop()
	}
}

// drop cancels a sync no longer listed and frees its buffer, once a step
// reading a page has finished.
func (j *syncJob) drop() {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.view.State = "canceled"
	j.archive.Records = nil
	j.known = nil
}

var syncPools = []string{"1", "2", "11", "12", "21", "22"}

func (s *Syncs) Start(store *Store, choice SyncChoice, uid, region string, full bool) (SyncInfo, error) {
	if !choice.Link && (choice.AccountRef == "" || choice.RoleRef == "") {
		return SyncInfo{}, ErrSync
	}
	archive, version, err := store.Snapshot(uid, region)
	if err != nil {
		return SyncInfo{}, err
	}
	incoming := Archive{UID: uid, Region: region, Timezone: 8, Language: "zh-cn", Records: []Record{}}
	if Validate(incoming) != nil || len(syncPools) == 0 {
		return SyncInfo{}, ErrInvalid
	}
	known := map[string]Record{}
	for _, r := range archive.Records {
		known[r.GachaType+":"+r.ID] = r
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.jobs == nil {
		s.jobs = map[string]*syncJob{}
	}
	now := time.Now()
	for ref, job := range s.jobs {
		if job.expired(now) {
			delete(s.jobs, ref)
		}
	}
	if len(s.jobs) >= 8 {
		return SyncInfo{}, ErrSync
	}
	ref := rand.Text()
	view := SyncInfo{Ref: ref, State: "running", Pool: syncPools[0]}
	job := &syncJob{view: view, choice: choice, archive: incoming, version: version, full: full, known: known, pools: syncPools, page: 1, endID: "0"}
	job.extend(now)
	s.jobs[ref] = job
	return view, nil
}
func (s *Syncs) job(ref string) (*syncJob, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	job := s.jobs[ref]
	if job == nil {
		return nil, ErrSync
	}
	return job, nil
}
func (s *Syncs) Choice(ref string) (SyncChoice, error) {
	job, err := s.job(ref)
	if err != nil {
		return SyncChoice{}, err
	}
	job.mu.Lock()
	defer job.mu.Unlock()
	if job.expired(time.Now()) {
		return SyncChoice{}, ErrSync
	}
	return job.choice, nil
}

func (s *Syncs) Info(ref string) (SyncInfo, error) {
	job, err := s.job(ref)
	if err != nil {
		return SyncInfo{}, err
	}
	job.mu.Lock()
	defer job.mu.Unlock()
	if job.expired(time.Now()) || job.view.State == "canceled" {
		return SyncInfo{}, ErrSync
	}
	return job.view, nil
}

// Forget drops a sync its caller reads to the end in one go, freeing its
// buffer and slot; the page's step-by-step syncs keep their replay API.
func (s *Syncs) Forget(ref string) {
	s.mu.Lock()
	job := s.jobs[ref]
	delete(s.jobs, ref)
	s.mu.Unlock()
	if job != nil {
		job.drop()
	}
}
func (s *Syncs) Cancel(ref string) error {
	job, err := s.job(ref)
	if err != nil {
		return err
	}
	job.mu.Lock()
	defer job.mu.Unlock()
	if job.view.State != "completed" {
		job.view.State = "canceled"
		job.archive.Records = nil
		job.known = nil
	}
	return nil
}
func (s *Syncs) Step(ctx context.Context, store *Store, ref string, sequence int, fetch FetchPage) (SyncInfo, error) {
	job, err := s.job(ref)
	if err != nil {
		return SyncInfo{}, err
	}
	job.mu.Lock()
	defer job.mu.Unlock()
	if job.expired(time.Now()) || job.view.State == "canceled" {
		return SyncInfo{}, ErrSync
	}
	if sequence == job.view.Sequence-1 {
		return job.view, nil
	}
	if sequence != job.view.Sequence || job.view.State != "running" {
		return SyncInfo{}, ErrSync
	}
	if err := ctx.Err(); err != nil {
		return SyncInfo{}, err
	}
	// Fetch owns only this event's caller; never retain it after this step.
	page, err := fetch(ctx, job.pools[job.pool], job.endID, job.page)
	if err != nil {
		return SyncInfo{}, err
	}
	if page.Timezone < -12 || page.Timezone > 14 || page.Language != "zh-cn" || len(page.Records) > 20 || page.More && (len(page.Records) == 0 || page.NextID == job.endID || page.NextID == "0") {
		return SyncInfo{}, ErrInvalid
	}
	if len(job.archive.Records) > 0 && len(page.Records) > 0 && job.archive.Timezone != page.Timezone {
		return SyncInfo{}, ErrConflict
	}
	batch := job.archive
	if len(page.Records) > 0 {
		batch.Timezone = page.Timezone
	}
	batch.Records = page.Records
	if err := Validate(batch); err != nil {
		return SyncInfo{}, err
	}
	if len(job.archive.Records)+len(page.Records) > 200000 || job.view.Pages >= 10000 {
		return SyncInfo{}, ErrInvalid
	}
	stop := !page.More
	for _, r := range page.Records {
		if old, exists := job.known[r.GachaType+":"+r.ID]; exists {
			if old.ItemID != r.ItemID || old.Time != r.Time || old.GachaType != r.GachaType || old.GachaID != r.GachaID {
				return SyncInfo{}, ErrConflict
			}
			if !job.full {
				stop = true
			}
		}
	}
	// Build candidate state first so a canceled request or failed final write
	// can retry the exact same page without duplicating its buffered records.
	if err := ctx.Err(); err != nil {
		return SyncInfo{}, err
	}
	if stop && job.pool+1 == len(job.pools) {
		candidate := job.archive
		if len(page.Records) > 0 {
			candidate.Timezone = page.Timezone
		}
		if len(job.archive.Records) == 0 && len(page.Records) == 0 {
			previous, _, err := store.Snapshot(job.archive.UID, job.archive.Region)
			if err != nil {
				return SyncInfo{}, err
			}
			if previous.UID != "" {
				candidate.Timezone = previous.Timezone
			}
		}
		candidate.Records = append(append([]Record{}, job.archive.Records...), page.Records...)
		merged, added, err := store.ImportIfUnchanged(candidate, job.version)
		if err != nil {
			return SyncInfo{}, err
		}
		job.view.State = "completed"
		job.view.Result = &ImportResult{UID: merged.UID, Region: merged.Region, Added: added, Total: len(merged.Records), Revision: merged.Revision}
		job.archive.Records = nil
		job.known = nil
	} else {
		if len(page.Records) > 0 {
			job.archive.Timezone = page.Timezone
		}
		job.archive.Records = append(job.archive.Records, page.Records...)
		if stop {
			job.pool++
			job.page = 1
			job.endID = "0"
		} else {
			job.page++
			job.endID = page.NextID
		}
		job.view.Pool = job.pools[job.pool]
	}
	job.view.Sequence++
	job.view.Pages++
	job.view.Fetched += len(page.Records)
	job.extend(time.Now())
	return job.view, nil
}
