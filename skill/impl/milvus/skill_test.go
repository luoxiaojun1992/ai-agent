package milvus

import (
	"context"
	"errors"
	"strings"
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

func TestMilvusSkill_Descriptions(t *testing.T) {
	i := &Insert{}
	s := &Search{}
	iDesc, err := i.GetDescription()
	if err != nil || iDesc == "" || i.ShortDescription() == "" {
		t.Fatalf("insert descriptions should not be empty")
	}
	sDesc, err := s.GetDescription()
	if err != nil || sDesc == "" || s.ShortDescription() == "" {
		t.Fatalf("search descriptions should not be empty")
	}
	if !strings.Contains(sDesc, "Parameters") {
		t.Fatalf("expected detailed search description")
	}
}

func TestInsert_Do_InvalidParams(t *testing.T) {
	i := &Insert{MilvusCli: &mockMilvusClient{}}
	if err := i.Do(context.Background(), "bad", nil); err == nil {
		t.Fatalf("expected invalid params error")
	}
}

func TestSearch_Do_InvalidParams(t *testing.T) {
	s := &Search{MilvusCli: &mockMilvusClient{}}
	if err := s.Do(context.Background(), "bad", nil); err == nil {
		t.Fatalf("expected invalid params error")
	}
}

func TestInsert_Do_MissingAndBadParams(t *testing.T) {
	i := &Insert{MilvusCli: &mockMilvusClient{}}
	// missing collection
	if err := i.Do(context.Background(), map[string]any{"content": "x", "vector": []float32{1.0}}, nil); err == nil {
		t.Fatalf("expected missing collection error")
	}
	// collection not string
	if err := i.Do(context.Background(), map[string]any{"collection": 1, "content": "x", "vector": []float32{1.0}}, nil); err == nil {
		t.Fatalf("expected collection type error")
	}
	// missing content
	if err := i.Do(context.Background(), map[string]any{"collection": "c", "vector": []float32{1.0}}, nil); err == nil {
		t.Fatalf("expected missing content error")
	}
	// content not string
	if err := i.Do(context.Background(), map[string]any{"collection": "c", "content": 1, "vector": []float32{1.0}}, nil); err == nil {
		t.Fatalf("expected content type error")
	}
	// missing vector
	if err := i.Do(context.Background(), map[string]any{"collection": "c", "content": "x"}, nil); err == nil {
		t.Fatalf("expected missing vector error")
	}
	// default bad vector type
	if err := i.Do(context.Background(), map[string]any{"collection": "c", "content": "x", "vector": "bad"}, nil); err == nil {
		t.Fatalf("expected bad vector type error")
	}
}

func TestInsert_Do_DirectFloat32Vector(t *testing.T) {
	cli := &mockMilvusClient{}
	i := &Insert{MilvusCli: cli}
	err := i.Do(context.Background(), map[string]any{
		"collection": "c",
		"content":    "hello",
		"vector":     []float32{1.0, 2.0},
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cli.insertCalled {
		t.Fatalf("expected insert called")
	}
}

func TestInsert_Do_Float32ElementInInterface(t *testing.T) {
	cli := &mockMilvusClient{}
	i := &Insert{MilvusCli: cli}
	err := i.Do(context.Background(), map[string]any{
		"collection": "c",
		"content":    "hello",
		"vector":     []interface{}{float32(0.5), float64(0.3)},
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error with mixed float types: %v", err)
	}
}

func TestInsert_Do_MilvusError(t *testing.T) {
	cli := &mockMilvusClient{insertErr: errors.New("insert failed")}
	i := &Insert{MilvusCli: cli}
	err := i.Do(context.Background(), map[string]any{
		"collection": "c",
		"content":    "hello",
		"vector":     []float32{1.0},
	}, nil)
	if err == nil {
		t.Fatalf("expected milvus insert error")
	}
}

func TestSearch_Do_MissingAndBadParams(t *testing.T) {
	s := &Search{MilvusCli: &mockMilvusClient{}}
	// missing collection
	if err := s.Do(context.Background(), map[string]any{"vector": []float32{1.0}}, func(any) (any, error) { return nil, nil }); err == nil {
		t.Fatalf("expected missing collection error")
	}
	// collection not string
	if err := s.Do(context.Background(), map[string]any{"collection": 1, "vector": []float32{1.0}}, func(any) (any, error) { return nil, nil }); err == nil {
		t.Fatalf("expected collection type error")
	}
	// missing vector
	if err := s.Do(context.Background(), map[string]any{"collection": "c"}, func(any) (any, error) { return nil, nil }); err == nil {
		t.Fatalf("expected missing vector error")
	}
	// default bad vector type
	if err := s.Do(context.Background(), map[string]any{"collection": "c", "vector": "bad"}, func(any) (any, error) { return nil, nil }); err == nil {
		t.Fatalf("expected bad vector type error")
	}
}

func TestSearch_Do_InterfaceVectorWithFloat32(t *testing.T) {
	cli := &mockMilvusClient{searchResult: []string{"r"}}
	s := &Search{MilvusCli: cli}
	err := s.Do(context.Background(), map[string]any{
		"collection": "c",
		"vector":     []interface{}{float32(0.5), float64(0.3)},
	}, func(any) (any, error) { return nil, nil })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSearch_Do_InterfaceVectorBadElement(t *testing.T) {
	s := &Search{MilvusCli: &mockMilvusClient{}}
	err := s.Do(context.Background(), map[string]any{
		"collection": "c",
		"vector":     []interface{}{1.0, "bad"},
	}, func(any) (any, error) { return nil, nil })
	if err == nil {
		t.Fatalf("expected bad element error")
	}
}

func TestSearch_Do_CallbackError(t *testing.T) {
	cli := &mockMilvusClient{searchResult: []string{"r"}}
	s := &Search{MilvusCli: cli}
	expected := errors.New("cb error")
	err := s.Do(context.Background(), map[string]any{
		"collection": "c",
		"vector":     []float32{0.1},
	}, func(any) (any, error) { return nil, expected })
	if !errors.Is(err, expected) {
		t.Fatalf("expected callback error, got: %v", err)
	}
}
