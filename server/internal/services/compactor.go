package services

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
)

const (
	DefaultMaxUpdates = 100

	DefaultMaxSizeBytes = 256 * 1024
)

type CompactorConfig struct {
	WorkerURL string

	MaxUpdates int64

	MaxSizeBytes int64
}

type CompactorService struct {
	yjsService      *YjsService
	snapshotService *SnapshotService
	config          CompactorConfig
	httpClient      *http.Client
}

func NewCompactorService(
	yjsService *YjsService,
	snapshotService *SnapshotService,
	cfg CompactorConfig,
) *CompactorService {
	if cfg.WorkerURL == "" {
		cfg.WorkerURL = "http://localhost:3001"
	}
	if cfg.MaxUpdates == 0 {
		cfg.MaxUpdates = DefaultMaxUpdates
	}
	if cfg.MaxSizeBytes == 0 {
		cfg.MaxSizeBytes = DefaultMaxSizeBytes
	}

	return &CompactorService{
		yjsService:      yjsService,
		snapshotService: snapshotService,
		config:          cfg,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (c *CompactorService) MaybeCompact(documentID uuid.UUID) error {
	triggered, reason, err := c.shouldCompact(documentID)
	if err != nil {
		return fmt.Errorf("compactor: threshold check failed: %w", err)
	}
	if !triggered {
		return nil
	}

	log.Printf("[Compactor] threshold reached (%s) for doc=%s — compacting", reason, documentID)
	return c.compact(documentID)
}

func (c *CompactorService) ForceCompact(documentID uuid.UUID) error {
	count, err := c.yjsService.GetDocumentUpdateCount(documentID)
	if err != nil {
		return err
	}
	if count == 0 {
		return nil
	}

	log.Printf("[Compactor] idle compaction triggered for doc=%s (%d updates)", documentID, count)
	return c.compact(documentID)
}

func (c *CompactorService) shouldCompact(documentID uuid.UUID) (bool, string, error) {
	count, err := c.yjsService.GetDocumentUpdateCount(documentID)
	if err != nil {
		return false, "", err
	}
	if count >= c.config.MaxUpdates {
		return true, fmt.Sprintf("count=%d >= %d", count, c.config.MaxUpdates), nil
	}

	size, err := c.yjsService.GetDocumentUpdateSize(documentID)
	if err != nil {
		return false, "", err
	}
	if size >= c.config.MaxSizeBytes {
		return true, fmt.Sprintf("size=%d >= %d bytes", size, c.config.MaxSizeBytes), nil
	}

	return false, "", nil
}

func (c *CompactorService) compact(documentID uuid.UUID) error {
	updates, err := c.yjsService.GetUpdates(documentID)
	if err != nil {
		return fmt.Errorf("compactor: failed to fetch updates: %w", err)
	}
	if len(updates) == 0 {
		return nil
	}

	var maxLamport int64
	rawUpdates := make([][]byte, len(updates))
	for i, u := range updates {
		rawUpdates[i] = u.Update
		if u.LamportTS > maxLamport {
			maxLamport = u.LamportTS
		}
	}

	merged, err := c.callWorker(rawUpdates)
	if err != nil {
		return fmt.Errorf("compactor: worker call failed: %w", err)
	}

	if err := c.snapshotService.SaveSnapshotAndClearUpdates(documentID, merged, maxLamport); err != nil {
		return fmt.Errorf("compactor: failed to save snapshot: %w", err)
	}

	log.Printf("[Compactor] compacted %d updates into snapshot (lamport=%d) for doc=%s",
		len(updates), maxLamport, documentID)
	return nil
}

type compactRequest struct {
	Updates [][]byte `json:"updates"`
}

type compactResponse struct {
	Snapshot string `json:"snapshot"`
	Error    string `json:"error,omitempty"`
}

func (c *CompactorService) callWorker(updates [][]byte) ([]byte, error) {
	body, err := json.Marshal(compactRequest{Updates: updates})
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := c.httpClient.Post(
		c.config.WorkerURL+"/compact",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf("http post: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("worker returned status %d: %s", resp.StatusCode, respBody)
	}

	var result compactResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	if result.Error != "" {
		return nil, fmt.Errorf("worker error: %s", result.Error)
	}

	merged, err := base64.StdEncoding.DecodeString(result.Snapshot)
	if err != nil {
		return nil, fmt.Errorf("base64 decode: %w", err)
	}

	return merged, nil
}

type stateVectorRequest struct {
	Snapshot string `json:"snapshot"`
}

type stateVectorResponse struct {
	StateVector string `json:"stateVector"`
	Error       string `json:"error,omitempty"`
}

func (c *CompactorService) GetStateVector(snapshot []byte) ([]byte, error) {
	encoded := base64.StdEncoding.EncodeToString(snapshot)

	body, err := json.Marshal(stateVectorRequest{Snapshot: encoded})
	if err != nil {
		return nil, fmt.Errorf("marshal state-vector request: %w", err)
	}

	resp, err := c.httpClient.Post(
		c.config.WorkerURL+"/state-vector",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf("http post state-vector: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read state-vector response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("worker returned status %d: %s", resp.StatusCode, respBody)
	}

	var result stateVectorResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal state-vector response: %w", err)
	}
	if result.Error != "" {
		return nil, fmt.Errorf("worker error: %s", result.Error)
	}

	sv, err := base64.StdEncoding.DecodeString(result.StateVector)
	if err != nil {
		return nil, fmt.Errorf("base64 decode state vector: %w", err)
	}

	return sv, nil
}

func (c *CompactorService) compactWithRetry(documentID uuid.UUID, maxRetries int) error {
	var lastErr error
	for i := range maxRetries {
		if err := c.compact(documentID); err == nil {
			return nil
		} else {
			lastErr = err
			wait := time.Duration(i+1) * 2 * time.Second
			log.Printf("[Compactor] retry %d/%d for doc=%s after %s: %v",
				i+1, maxRetries, documentID, wait, err)
			time.Sleep(wait)
		}
	}
	return fmt.Errorf("compaction failed after %d retries: %w", maxRetries, lastErr)
}

func (c *CompactorService) MaybeCompactWithRetry(documentID uuid.UUID) error {
	triggered, reason, err := c.shouldCompact(documentID)
	if err != nil {
		return fmt.Errorf("compactor: threshold check failed: %w", err)
	}
	if !triggered {
		return nil
	}
	log.Printf("[Compactor] threshold reached (%s) for doc=%s — compacting (with retry)", reason, documentID)
	return c.compactWithRetry(documentID, 3)
}

type snapshotRequest struct {
	Updates [][]byte `json:"updates"`
}

type snapshotWorkerResponse struct {
	Snapshot string `json:"snapshot"`
	Error    string `json:"error,omitempty"`
}

func (c *CompactorService) GetSemanticSnapshot(documentID uuid.UUID) ([]byte, error) {
	updates, err := c.yjsService.GetUpdates(documentID)
	if err != nil {
		return nil, fmt.Errorf("GetSemanticSnapshot: fetch updates: %w", err)
	}
	if len(updates) == 0 {
		return nil, nil
	}

	rawUpdates := make([][]byte, len(updates))
	for i, u := range updates {
		rawUpdates[i] = u.Update
	}

	body, err := json.Marshal(snapshotRequest{Updates: rawUpdates})
	if err != nil {
		return nil, fmt.Errorf("GetSemanticSnapshot: marshal: %w", err)
	}

	resp, err := c.httpClient.Post(
		c.config.WorkerURL+"/snapshot",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf("GetSemanticSnapshot: http post: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("GetSemanticSnapshot: read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GetSemanticSnapshot: worker status %d: %s", resp.StatusCode, respBody)
	}

	var result snapshotWorkerResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("GetSemanticSnapshot: unmarshal: %w", err)
	}
	if result.Error != "" {
		return nil, fmt.Errorf("GetSemanticSnapshot: worker error: %s", result.Error)
	}

	snap, err := base64.StdEncoding.DecodeString(result.Snapshot)
	if err != nil {
		return nil, fmt.Errorf("GetSemanticSnapshot: base64 decode: %w", err)
	}
	return snap, nil
}

type diffRequest struct {
	Updates     [][]byte `json:"updates"`
	StateVector string   `json:"stateVector"`
}

type diffResponse struct {
	Diff  string `json:"diff"`
	Error string `json:"error,omitempty"`
}

func (c *CompactorService) GetDiff(documentID uuid.UUID, clientStateVector []byte) ([]byte, error) {
	updates, err := c.yjsService.GetUpdates(documentID)
	if err != nil {
		return nil, fmt.Errorf("GetDiff: fetch updates: %w", err)
	}

	rawUpdates := make([][]byte, len(updates))
	for i, u := range updates {
		rawUpdates[i] = u.Update
	}

	svEncoded := base64.StdEncoding.EncodeToString(clientStateVector)
	body, err := json.Marshal(diffRequest{Updates: rawUpdates, StateVector: svEncoded})
	if err != nil {
		return nil, fmt.Errorf("GetDiff: marshal: %w", err)
	}

	resp, err := c.httpClient.Post(
		c.config.WorkerURL+"/diff",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf("GetDiff: http post: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("GetDiff: read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GetDiff: worker status %d: %s", resp.StatusCode, respBody)
	}

	var result diffResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("GetDiff: unmarshal: %w", err)
	}
	if result.Error != "" {
		return nil, fmt.Errorf("GetDiff: worker error: %s", result.Error)
	}

	diff, err := base64.StdEncoding.DecodeString(result.Diff)
	if err != nil {
		return nil, fmt.Errorf("GetDiff: base64 decode: %w", err)
	}
	return diff, nil
}
