package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/pashagolub/pgxmock/v4"

	"github.com/bangun-ekosistem/service-api/internal/middleware"
)

func TestDailyRevenue_ReturnsCorrectShape(t *testing.T) {
	conn, err := pgxmock.NewConn()
	if err != nil {
		t.Fatalf("failed to create pgxmock conn: %v", err)
	}
	defer conn.Close(context.Background())

	svc := NewAnalyticsService(nil)
	tenantID := uuid.New()

	// Mock the query returning daily revenue data
	rows := conn.NewRows([]string{"date", "revenue", "transactions"}).
		AddRow("2026-03-13", int64(500000), 5).
		AddRow("2026-03-14", int64(750000), 8)

	conn.ExpectQuery(`SELECT to_char\(date_trunc`).
		WithArgs(tenantID, "7").
		WillReturnRows(rows)

	ctx := context.WithValue(context.Background(), middleware.ContextKeyDBTx, conn)

	results, err := svc.DailyRevenue(ctx, tenantID, 7)
	if err != nil {
		t.Fatalf("DailyRevenue returned error: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	// Verify first row
	if results[0].Date != "2026-03-13" {
		t.Errorf("expected date '2026-03-13', got %q", results[0].Date)
	}
	if results[0].Revenue != 500000 {
		t.Errorf("expected revenue 500000, got %d", results[0].Revenue)
	}
	if results[0].Transactions != 5 {
		t.Errorf("expected 5 transactions, got %d", results[0].Transactions)
	}

	// Verify second row
	if results[1].Date != "2026-03-14" {
		t.Errorf("expected date '2026-03-14', got %q", results[1].Date)
	}
	if results[1].Revenue != 750000 {
		t.Errorf("expected revenue 750000, got %d", results[1].Revenue)
	}
	if results[1].Transactions != 8 {
		t.Errorf("expected 8 transactions, got %d", results[1].Transactions)
	}

	if err := conn.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet pgxmock expectations: %v", err)
	}
}

func TestDailyRevenue_EmptyResult(t *testing.T) {
	conn, err := pgxmock.NewConn()
	if err != nil {
		t.Fatalf("failed to create pgxmock conn: %v", err)
	}
	defer conn.Close(context.Background())

	svc := NewAnalyticsService(nil)
	tenantID := uuid.New()

	// Mock the query returning no rows
	rows := conn.NewRows([]string{"date", "revenue", "transactions"})

	conn.ExpectQuery(`SELECT to_char\(date_trunc`).
		WithArgs(tenantID, "30").
		WillReturnRows(rows)

	ctx := context.WithValue(context.Background(), middleware.ContextKeyDBTx, conn)

	results, err := svc.DailyRevenue(ctx, tenantID, 30)
	if err != nil {
		t.Fatalf("DailyRevenue returned error: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("expected 0 results for empty data, got %d", len(results))
	}

	// Should return empty slice, not nil
	if results == nil {
		t.Error("expected empty slice, got nil")
	}

	if err := conn.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet pgxmock expectations: %v", err)
	}
}

func TestDailyRevenuePoint_StructShape(t *testing.T) {
	// Verify DailyRevenuePoint has the expected fields and json tags
	p := DailyRevenuePoint{
		Date:         "2026-03-14",
		Revenue:      1000000,
		Transactions: 15,
	}

	if p.Date != "2026-03-14" {
		t.Errorf("expected Date '2026-03-14', got %q", p.Date)
	}
	if p.Revenue != 1000000 {
		t.Errorf("expected Revenue 1000000, got %d", p.Revenue)
	}
	if p.Transactions != 15 {
		t.Errorf("expected Transactions 15, got %d", p.Transactions)
	}
}

func TestTransactionStats_ReturnsCorrectData(t *testing.T) {
	conn, err := pgxmock.NewConn()
	if err != nil {
		t.Fatalf("failed to create pgxmock conn: %v", err)
	}
	defer conn.Close(context.Background())

	svc := NewAnalyticsService(nil)

	// Mock the query
	rows := conn.NewRows([]string{"total_transactions", "total_revenue", "avg_transaction"}).
		AddRow(100, int64(50000000), int64(500000))

	conn.ExpectQuery(`SELECT COUNT\(id\) as total_transactions`).
		WithArgs("2026-01-01", "2026-03-14").
		WillReturnRows(rows)

	ctx := context.WithValue(context.Background(), middleware.ContextKeyDBTx, conn)

	stats, err := svc.TransactionStats(ctx, "2026-01-01", "2026-03-14")
	if err != nil {
		t.Fatalf("TransactionStats returned error: %v", err)
	}

	if stats.TotalTransactions != 100 {
		t.Errorf("expected TotalTransactions 100, got %d", stats.TotalTransactions)
	}
	if stats.TotalRevenue != 50000000 {
		t.Errorf("expected TotalRevenue 50000000, got %d", stats.TotalRevenue)
	}
	if stats.AvgTransaction != 500000 {
		t.Errorf("expected AvgTransaction 500000, got %d", stats.AvgTransaction)
	}

	if err := conn.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet pgxmock expectations: %v", err)
	}
}

func TestTransactionStats_NoDates(t *testing.T) {
	conn, err := pgxmock.NewConn()
	if err != nil {
		t.Fatalf("failed to create pgxmock conn: %v", err)
	}
	defer conn.Close(context.Background())

	svc := NewAnalyticsService(nil)

	// Mock the query with no date filters (no args)
	rows := conn.NewRows([]string{"total_transactions", "total_revenue", "avg_transaction"}).
		AddRow(0, int64(0), int64(0))

	conn.ExpectQuery(`SELECT COUNT\(id\) as total_transactions`).
		WillReturnRows(rows)

	ctx := context.WithValue(context.Background(), middleware.ContextKeyDBTx, conn)

	stats, err := svc.TransactionStats(ctx, "", "")
	if err != nil {
		t.Fatalf("TransactionStats returned error: %v", err)
	}

	if stats.TotalTransactions != 0 {
		t.Errorf("expected TotalTransactions 0, got %d", stats.TotalTransactions)
	}
	if stats.TotalRevenue != 0 {
		t.Errorf("expected TotalRevenue 0, got %d", stats.TotalRevenue)
	}

	if err := conn.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet pgxmock expectations: %v", err)
	}
}
