package services

import (
	"github.com/google/uuid"
	"github.com/guijoazeiro/text-editor/tree/main/server/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SnapshotService struct {
	db         *gorm.DB
	yjsService *YjsService
}

func NewSnapshotService(db *gorm.DB, yjsService *YjsService) *SnapshotService {
	return &SnapshotService{db: db, yjsService: yjsService}
}

type ClientState struct {
	Snapshot []byte

	SnapshotLamport int64

	DeltaUpdates [][]byte
}

func (s *SnapshotService) GetStateForClient(documentID uuid.UUID) (*ClientState, error) {
	snap, err := s.GetSnapshot(documentID)
	if err != nil {
		return nil, err
	}

	var deltaUpdates [][]byte
	var sinceLamport int64

	if snap != nil {
		sinceLamport = snap.LamportTS
	}

	updates, err := s.yjsService.GetUpdatesSince(documentID, sinceLamport)
	if err != nil {
		return nil, err
	}
	for _, u := range updates {
		deltaUpdates = append(deltaUpdates, u.Update)
	}

	if snap == nil {
		updates, err = s.yjsService.GetUpdates(documentID)
		if err != nil {
			return nil, err
		}
		deltaUpdates = make([][]byte, 0, len(updates))
		for _, u := range updates {
			deltaUpdates = append(deltaUpdates, u.Update)
		}
	}

	state := &ClientState{
		DeltaUpdates: deltaUpdates,
	}
	if snap != nil {
		state.Snapshot = snap.Snapshot
		state.SnapshotLamport = snap.LamportTS
	}
	return state, nil
}

func (s *SnapshotService) GetSnapshot(documentID uuid.UUID) (*models.YjsSnapshot, error) {
	var snap models.YjsSnapshot
	err := s.db.Where("document_id = ?", documentID).First(&snap).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &snap, nil
}

func (s *SnapshotService) SaveSnapshotAndClearUpdates(
	documentID uuid.UUID,
	snapshot []byte,
	lamportTS int64,
) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		snap := models.YjsSnapshot{
			DocumentID: documentID,
			Snapshot:   snapshot,
			LamportTS:  lamportTS,
		}

		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "document_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"snapshot", "lamport_ts", "created_at"}),
		}).Create(&snap).Error; err != nil {
			return err
		}

		return tx.Where("document_id = ?", documentID).
			Delete(&models.YjsUpdate{}).Error
	})
}
