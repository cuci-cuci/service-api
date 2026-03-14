package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/pashagolub/pgxmock/v4"

	"github.com/bangun-ekosistem/service-api/internal/domain"
	"github.com/bangun-ekosistem/service-api/internal/middleware"
)

func TestRecordMovement_StockIn(t *testing.T) {
	conn, err := pgxmock.NewConn()
	if err != nil {
		t.Fatalf("failed to create pgxmock conn: %v", err)
	}
	defer conn.Close(context.Background())

	svc := NewInventoryService(nil)
	tenantID := uuid.New()
	userID := uuid.New()
	supplyID := uuid.New()

	req := domain.StockMovementRequest{
		SupplyID:     supplyID.String(),
		MovementType: "in",
		Quantity:     10.0,
		Notes:        "Restocking supplies",
	}

	// Expect INSERT into stock_movements
	conn.ExpectExec(`INSERT INTO stock_movements`).
		WithArgs(tenantID, supplyID, "in", 10.0, pgxmock.AnyArg(), userID).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	// Expect UPDATE supplies with positive delta (+10.0)
	conn.ExpectExec(`UPDATE supplies SET current_stock = current_stock \+ \$1`).
		WithArgs(10.0, supplyID, tenantID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	ctx := context.WithValue(context.Background(), middleware.ContextKeyDBTx, conn)

	err = svc.RecordMovement(ctx, tenantID, userID, req)
	if err != nil {
		t.Fatalf("RecordMovement (stock_in) returned error: %v", err)
	}

	if err := conn.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet pgxmock expectations: %v", err)
	}
}

func TestRecordMovement_StockOut(t *testing.T) {
	conn, err := pgxmock.NewConn()
	if err != nil {
		t.Fatalf("failed to create pgxmock conn: %v", err)
	}
	defer conn.Close(context.Background())

	svc := NewInventoryService(nil)
	tenantID := uuid.New()
	userID := uuid.New()
	supplyID := uuid.New()

	req := domain.StockMovementRequest{
		SupplyID:     supplyID.String(),
		MovementType: "out",
		Quantity:     5.0,
		Notes:        "Used for order",
	}

	// Expect INSERT into stock_movements
	conn.ExpectExec(`INSERT INTO stock_movements`).
		WithArgs(tenantID, supplyID, "out", 5.0, pgxmock.AnyArg(), userID).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	// Expect UPDATE supplies with negative delta (-5.0)
	conn.ExpectExec(`UPDATE supplies SET current_stock = current_stock \+ \$1`).
		WithArgs(-5.0, supplyID, tenantID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	ctx := context.WithValue(context.Background(), middleware.ContextKeyDBTx, conn)

	err = svc.RecordMovement(ctx, tenantID, userID, req)
	if err != nil {
		t.Fatalf("RecordMovement (stock_out) returned error: %v", err)
	}

	if err := conn.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet pgxmock expectations: %v", err)
	}
}

func TestRecordMovement_Adjustment(t *testing.T) {
	conn, err := pgxmock.NewConn()
	if err != nil {
		t.Fatalf("failed to create pgxmock conn: %v", err)
	}
	defer conn.Close(context.Background())

	svc := NewInventoryService(nil)
	tenantID := uuid.New()
	userID := uuid.New()
	supplyID := uuid.New()

	req := domain.StockMovementRequest{
		SupplyID:     supplyID.String(),
		MovementType: "adjustment",
		Quantity:     25.0,
		Notes:        "Inventory count correction",
	}

	// Expect INSERT into stock_movements
	conn.ExpectExec(`INSERT INTO stock_movements`).
		WithArgs(tenantID, supplyID, "adjustment", 25.0, pgxmock.AnyArg(), userID).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	// Expect UPDATE supplies setting absolute value (not delta)
	conn.ExpectExec(`UPDATE supplies SET current_stock = \$1`).
		WithArgs(25.0, supplyID, tenantID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	ctx := context.WithValue(context.Background(), middleware.ContextKeyDBTx, conn)

	err = svc.RecordMovement(ctx, tenantID, userID, req)
	if err != nil {
		t.Fatalf("RecordMovement (adjustment) returned error: %v", err)
	}

	if err := conn.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet pgxmock expectations: %v", err)
	}
}

func TestRecordMovement_InvalidSupplyID(t *testing.T) {
	svc := NewInventoryService(nil)
	tenantID := uuid.New()
	userID := uuid.New()

	req := domain.StockMovementRequest{
		SupplyID:     "not-a-valid-uuid",
		MovementType: "in",
		Quantity:     10.0,
	}

	// No mock needed; validation happens before DB call
	ctx := context.Background()

	err := svc.RecordMovement(ctx, tenantID, userID, req)
	if err == nil {
		t.Fatal("expected error for invalid supply_id, got nil")
	}
	if err.Error() != "invalid supply_id" {
		t.Errorf("expected error message 'invalid supply_id', got %q", err.Error())
	}
}

func TestRecordMovement_StockOut_SupplyNotFound(t *testing.T) {
	conn, err := pgxmock.NewConn()
	if err != nil {
		t.Fatalf("failed to create pgxmock conn: %v", err)
	}
	defer conn.Close(context.Background())

	svc := NewInventoryService(nil)
	tenantID := uuid.New()
	userID := uuid.New()
	supplyID := uuid.New()

	req := domain.StockMovementRequest{
		SupplyID:     supplyID.String(),
		MovementType: "out",
		Quantity:     5.0,
	}

	// Expect INSERT into stock_movements
	conn.ExpectExec(`INSERT INTO stock_movements`).
		WithArgs(tenantID, supplyID, "out", 5.0, pgxmock.AnyArg(), userID).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	// Expect UPDATE supplies but no rows affected (supply not found)
	conn.ExpectExec(`UPDATE supplies SET current_stock = current_stock \+ \$1`).
		WithArgs(-5.0, supplyID, tenantID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 0))

	ctx := context.WithValue(context.Background(), middleware.ContextKeyDBTx, conn)

	err = svc.RecordMovement(ctx, tenantID, userID, req)
	if err == nil {
		t.Fatal("expected error when supply not found, got nil")
	}
	if err.Error() != "supply not found" {
		t.Errorf("expected error message 'supply not found', got %q", err.Error())
	}

	if err := conn.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet pgxmock expectations: %v", err)
	}
}

func TestRecordMovement_StockIn_WithoutNotes(t *testing.T) {
	conn, err := pgxmock.NewConn()
	if err != nil {
		t.Fatalf("failed to create pgxmock conn: %v", err)
	}
	defer conn.Close(context.Background())

	svc := NewInventoryService(nil)
	tenantID := uuid.New()
	userID := uuid.New()
	supplyID := uuid.New()

	req := domain.StockMovementRequest{
		SupplyID:     supplyID.String(),
		MovementType: "in",
		Quantity:     3.5,
		Notes:        "", // Empty notes should pass nil to DB
	}

	// Expect INSERT into stock_movements - empty notes becomes (*string)(nil)
	conn.ExpectExec(`INSERT INTO stock_movements`).
		WithArgs(tenantID, supplyID, "in", 3.5, pgxmock.AnyArg(), userID).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	// Expect UPDATE supplies
	conn.ExpectExec(`UPDATE supplies SET current_stock = current_stock \+ \$1`).
		WithArgs(3.5, supplyID, tenantID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	ctx := context.WithValue(context.Background(), middleware.ContextKeyDBTx, conn)

	err = svc.RecordMovement(ctx, tenantID, userID, req)
	if err != nil {
		t.Fatalf("RecordMovement returned error: %v", err)
	}

	if err := conn.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet pgxmock expectations: %v", err)
	}
}
