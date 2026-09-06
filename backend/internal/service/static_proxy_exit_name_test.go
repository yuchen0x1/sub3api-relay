//go:build unit

package service

import (
	"context"
	"testing"
)

type stubExitProxyRepo struct {
	ProxyRepository
	proxy *Proxy
}

func (s *stubExitProxyRepo) GetByID(context.Context, int64) (*Proxy, error) {
	return s.proxy, nil
}

type stubExitRepo struct {
	StaticProxyExitRepository
	created *StaticProxyExit
}

func (s *stubExitRepo) Create(_ context.Context, exit *StaticProxyExit) error {
	exit.ID = 1
	s.created = exit
	return nil
}

func TestStaticProxyExitCreateGeneratesDefaultName(t *testing.T) {
	proxy := &Proxy{ID: 7, Name: "us美国高速05", Host: "cfyes.example.xyz"}
	svc := &StaticProxyExitService{
		repo:      &stubExitRepo{},
		proxyRepo: &stubExitProxyRepo{proxy: proxy},
	}

	exit, err := svc.Create(context.Background(), StaticProxyExitCreateInput{
		AccountID: 42,
		ProxyID:   7,
		Name:      "",
		Position:  3,
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if exit.Name != "us美国高速05-3" {
		t.Fatalf("expected default name 'us美国高速05-3', got %q", exit.Name)
	}
}

func TestStaticProxyExitCreateFallsBackToHost(t *testing.T) {
	proxy := &Proxy{ID: 7, Name: "  ", Host: "cfyes.example.xyz"}
	svc := &StaticProxyExitService{
		repo:      &stubExitRepo{},
		proxyRepo: &stubExitProxyRepo{proxy: proxy},
	}

	exit, err := svc.Create(context.Background(), StaticProxyExitCreateInput{
		AccountID: 42,
		ProxyID:   7,
		Name:      "",
		Position:  0,
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if exit.Name != "cfyes.example.xyz" {
		t.Fatalf("expected host fallback name, got %q", exit.Name)
	}
}
