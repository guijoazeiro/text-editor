package yjs

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

type UpdateMeta struct {
	ClientID int64

	LamportTS int64

	IsEmpty bool
}

func DecodeUpdateMeta(data []byte) (*UpdateMeta, error) {
	if len(data) < 2 {
		return nil, fmt.Errorf("yjs: update too short (%d bytes, minimum 2)", len(data))
	}

	if data[0] == 0x00 {
		return &UpdateMeta{IsEmpty: true}, nil
	}

	r := bytes.NewReader(data)

	numClients, err := binary.ReadUvarint(r)
	if err != nil {
		return nil, fmt.Errorf("yjs: failed to read numClients: %w", err)
	}

	if numClients == 0 {
		return &UpdateMeta{IsEmpty: true}, nil
	}

	if numClients > 10_000 {
		return nil, fmt.Errorf("yjs: implausible numClients value %d — possible corrupt data", numClients)
	}

	clientID, err := binary.ReadUvarint(r)
	if err != nil {
		return nil, fmt.Errorf("yjs: failed to read clientID: %w", err)
	}

	numStructs, err := binary.ReadUvarint(r)
	if err != nil {
		return nil, fmt.Errorf("yjs: failed to read numStructs: %w", err)
	}

	startClock, err := binary.ReadUvarint(r)
	if err != nil {
		return nil, fmt.Errorf("yjs: failed to read startClock: %w", err)
	}

	var lamportTS uint64
	if numStructs > 0 {
		lamportTS = startClock + numStructs - 1
	} else {
		lamportTS = startClock
	}

	return &UpdateMeta{
		ClientID:  int64(clientID),
		LamportTS: int64(lamportTS),
		IsEmpty:   false,
	}, nil
}

func ValidateUpdate(data []byte) error {
	_, err := DecodeUpdateMeta(data)
	return err
}
