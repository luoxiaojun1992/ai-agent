package milvus

import (
	"context"
	"errors"
	"testing"
)

type mockMilvusClient struct {
	insertErr error
	searchErr error

	insertCalled bool
	searchCalled bool

	insertCollection string
	insertContent    string
	insertVector     []float32

	searchCollection string
	searchVector     []float32
	searchResult     []string
}

func (m *mockMilvusClient) InsertVector(ctx context.Context, collectionName, content string, vector []float32) error {
	_ = ctx
	m.insertCalled = true
	m.insertCollection = collectionName
	m.insertContent = content
	m.insertVector = vector
	return m.insertErr
}

func (m *mockMilvusClient) SearchVector(ctx context.Context, collectionName string, vector []float32) ([]string, error) {
	_ = ctx
	m.searchCalled = true
	m.searchCollection = collectionName
	m.searchVector = vector
	if m.searchErr != nil {
		return nil, m.searchErr
	}
	return m.searchResult, nil
}

func (m *mockMilvusClient) Close() error { return nil }

func TestInsert_Do_SuccessWithInterfaceVector(t *testing.T) {
	cli := &mockMilvusClient{}
	s := &Insert{MilvusCli: cli}

	err := s.Do(context.Background(), map[string]any{
		"collection": "c",
		"content":    "hello",
		"vector":     []interface{}{1.0, 2.0},
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cli.insertCalled || cli.insertCollection != "c" || cli.insertContent != "hello" || len(cli.insertVector) != 2 {
		t.Fatalf("unexpected insert capture: %+v", cli)
	}
}

func TestInsert_Do_InvalidVectorElement(t *testing.T) {
	s := &Insert{MilvusCli: &mockMilvusClient{}}
	err := s.Do(context.Background(), map[string]any{
		"collection": "c",
		"content":    "hello",
		"vector":     []interface{}{1.0, "bad"},
	}, nil)
	if err == nil {
		t.Fatalf("expected vector element conversion error")
	}
}

func TestSearch_Do_Success(t *testing.T) {
	cli := &mockMilvusClient{searchResult: []string{"a", "b"}}
	s := &Search{MilvusCli: cli}

	callbackCalled := false
	err := s.Do(context.Background(), map[string]any{
		"collection": "c",
		"vector":     []float32{0.1, 0.2},
	}, func(output any) (any, error) {
		callbackCalled = true
		items, ok := output.([]string)
		if !ok || len(items) != 2 {
			t.Fatalf("unexpected callback output: %#v", output)
		}
		return nil, nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !callbackCalled || !cli.searchCalled {
		t.Fatalf("expected callback and search call")
	}
}

func TestSearch_Do_SearchError(t *testing.T) {
	expected := errors.New("search failed")
	s := &Search{MilvusCli: &mockMilvusClient{searchErr: expected}}
	err := s.Do(context.Background(), map[string]any{
		"collection": "c",
		"vector":     []float32{0.1},
	}, func(output any) (any, error) { return nil, nil })
	if !errors.Is(err, expected) {
		t.Fatalf("expected search error, got: %v", err)
	}
}
