package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pashagolub/pgxmock/v4"

	"github.com/bangun-ekosistem/service-api/internal/domain"
	"github.com/bangun-ekosistem/service-api/internal/middleware"
)

// --- Pure logic tests for validTransitions ---

func TestValidTransitions_AllStatusesHaveEntries(t *testing.T) {
	expectedStatuses := []string{"received", "washing", "drying", "ironing", "done"}
	for _, status := range expectedStatuses {
		if _, ok := validTransitions[status]; !ok {
			t.Errorf("expected validTransitions to contain status %q", status)
		}
	}
}

func TestValidTransitions_TerminalStatusesHaveNoEntry(t *testing.T) {
	terminalStatuses := []string{"picked_up", "cancelled"}
	for _, status := range terminalStatuses {
		if _, ok := validTransitions[status]; ok {
			t.Errorf("terminal status %q should not be a key in validTransitions", status)
		}
	}
}

func TestValidTransitions_ReceivedCanTransitionToWashingOrCancelled(t *testing.T) {
	allowed := validTransitions["received"]
	expected := map[string]bool{"washing": true, "cancelled": true}

	if len(allowed) != len(expected) {
		t.Fatalf("expected %d transitions from 'received', got %d", len(expected), len(allowed))
	}
	for _, s := range allowed {
		if !expected[s] {
			t.Errorf("unexpected transition from 'received' to %q", s)
		}
	}
}

func TestValidTransitions_WashingCanTransitionCorrectly(t *testing.T) {
	allowed := validTransitions["washing"]
	expected := map[string]bool{"drying": true, "ironing": true, "cancelled": true}

	if len(allowed) != len(expected) {
		t.Fatalf("expected %d transitions from 'washing', got %d", len(expected), len(allowed))
	}
	for _, s := range allowed {
		if !expected[s] {
			t.Errorf("unexpected transition from 'washing' to %q", s)
		}
	}
}

func TestValidTransitions_DryingCanTransitionCorrectly(t *testing.T) {
	allowed := validTransitions["drying"]
	expected := map[string]bool{"ironing": true, "done": true, "cancelled": true}

	if len(allowed) != len(expected) {
		t.Fatalf("expected %d transitions from 'drying', got %d", len(expected), len(allowed))
	}
	for _, s := range allowed {
		if !expected[s] {
			t.Errorf("unexpected transition from 'drying' to %q", s)
		}
	}
}

func TestValidTransitions_IroningCanTransitionCorrectly(t *testing.T) {
	allowed := validTransitions["ironing"]
	expected := map[string]bool{"done": true, "cancelled": true}

	if len(allowed) != len(expected) {
		t.Fatalf("expected %d transitions from 'ironing', got %d", len(expected), len(allowed))
	}
	for _, s := range allowed {
		if !expected[s] {
			t.Errorf("unexpected transition from 'ironing' to %q", s)
		}
	}
}

func TestValidTransitions_DoneCanTransitionCorrectly(t *testing.T) {
	allowed := validTransitions["done"]
	expected := map[string]bool{"picked_up": true, "cancelled": true}

	if len(allowed) != len(expected) {
		t.Fatalf("expected %d transitions from 'done', got %d", len(expected), len(allowed))
	}
	for _, s := range allowed {
		if !expected[s] {
			t.Errorf("unexpected transition from 'done' to %q", s)
		}
	}
}

func TestValidTransitions_CancelledIsAlwaysAllowed(t *testing.T) {
	for status, transitions := range validTransitions {
		hasCancelled := false
		for _, target := range transitions {
			if target == "cancelled" {
				hasCancelled = true
				break
			}
		}
		if !hasCancelled {
			t.Errorf("status %q should allow transition to 'cancelled'", status)
		}
	}
}

// --- pgxmock-based tests for UpdateStatus ---

func TestUpdateStatus_ValidTransition(t *testing.T) {
	testCases := []struct {
		name       string
		fromStatus string
		toStatus   string
	}{
		{"received to washing", "received", "washing"},
		{"received to cancelled", "received", "cancelled"},
		{"washing to drying", "washing", "drying"},
		{"washing to ironing", "washing", "ironing"},
		{"drying to ironing", "drying", "ironing"},
		{"drying to done", "drying", "done"},
		{"ironing to done", "ironing", "done"},
		{"done to picked_up", "done", "picked_up"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			conn, err := pgxmock.NewConn()
			if err != nil {
				t.Fatalf("failed to create pgxmock conn: %v", err)
			}
			defer conn.Close(context.Background())

			orderID := uuid.New()
			tenantID := uuid.New()
			outletID := uuid.New()
			userID := uuid.New()
			now := time.Now()

			// Mock the SELECT query to get current order
			orderRows := conn.NewRows([]string{
				"id", "transaction_id", "tenant_id", "outlet_id", "status",
				"estimated_completion_at", "completed_at", "picked_up_at", "notes",
				"customer_phone", "tracking_token",
				"delivery_type", "delivery_address", "delivery_fee", "scheduled_pickup_at",
				"created_by", "updated_by", "created_at", "updated_at",
			}).AddRow(
				orderID, nil, tenantID, outletID, tc.fromStatus,
				nil, nil, nil, nil,
				nil, nil,
				"pickup", nil, int64(0), nil,
				userID, userID, now, now,
			)

			conn.ExpectQuery(`SELECT (.+) FROM orders WHERE id = \$1`).
				WithArgs(orderID).
				WillReturnRows(orderRows)

			// Mock the UPDATE query - args vary by target status
			updateArgs := []any{
				pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
			}
			if tc.toStatus == "done" || tc.toStatus == "picked_up" {
				updateArgs = append(updateArgs, pgxmock.AnyArg())
			}
			updateArgs = append(updateArgs, pgxmock.AnyArg())
			conn.ExpectExec(`UPDATE orders SET status`).
				WithArgs(updateArgs...).
				WillReturnResult(pgxmock.NewResult("UPDATE", 1))

			// Mock the INSERT into order_status_logs (7 args)
			conn.ExpectExec(`INSERT INTO order_status_logs`).
				WithArgs(
					pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
					pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
					pgxmock.AnyArg(),
				).
				WillReturnResult(pgxmock.NewResult("INSERT", 1))

			svc := NewOrderService(nil)
			ctx := context.WithValue(context.Background(), middleware.ContextKeyDBTx, conn)

			req := domain.UpdateOrderStatusRequest{
				Status: tc.toStatus,
			}

			result, err := svc.UpdateStatus(ctx, orderID, userID, req)
			if err != nil {
				t.Fatalf("UpdateStatus returned error for valid transition %s -> %s: %v", tc.fromStatus, tc.toStatus, err)
			}
			if result.Status != tc.toStatus {
				t.Errorf("expected status %s, got %s", tc.toStatus, result.Status)
			}

			if tc.toStatus == "done" && result.CompletedAt == nil {
				t.Error("expected CompletedAt to be set when status is 'done'")
			}
			if tc.toStatus == "picked_up" && result.PickedUpAt == nil {
				t.Error("expected PickedUpAt to be set when status is 'picked_up'")
			}

			if err := conn.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet pgxmock expectations: %v", err)
			}
		})
	}
}

func TestUpdateStatus_InvalidTransition(t *testing.T) {
	testCases := []struct {
		name       string
		fromStatus string
		toStatus   string
	}{
		{"received to done (skip steps)", "received", "done"},
		{"received to picked_up (skip all)", "received", "picked_up"},
		{"received to ironing (skip washing)", "received", "ironing"},
		{"washing to picked_up (skip done)", "washing", "picked_up"},
		{"done to washing (backwards)", "done", "washing"},
		{"ironing to washing (backwards)", "ironing", "washing"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			conn, err := pgxmock.NewConn()
			if err != nil {
				t.Fatalf("failed to create pgxmock conn: %v", err)
			}
			defer conn.Close(context.Background())

			orderID := uuid.New()
			tenantID := uuid.New()
			outletID := uuid.New()
			userID := uuid.New()
			now := time.Now()

			orderRows := conn.NewRows([]string{
				"id", "transaction_id", "tenant_id", "outlet_id", "status",
				"estimated_completion_at", "completed_at", "picked_up_at", "notes",
				"customer_phone", "tracking_token",
				"delivery_type", "delivery_address", "delivery_fee", "scheduled_pickup_at",
				"created_by", "updated_by", "created_at", "updated_at",
			}).AddRow(
				orderID, nil, tenantID, outletID, tc.fromStatus,
				nil, nil, nil, nil,
				nil, nil,
				"pickup", nil, int64(0), nil,
				userID, userID, now, now,
			)

			conn.ExpectQuery(`SELECT (.+) FROM orders WHERE id = \$1`).
				WithArgs(orderID).
				WillReturnRows(orderRows)

			svc := NewOrderService(nil)
			ctx := context.WithValue(context.Background(), middleware.ContextKeyDBTx, conn)

			req := domain.UpdateOrderStatusRequest{
				Status: tc.toStatus,
			}

			_, err = svc.UpdateStatus(ctx, orderID, userID, req)
			if err == nil {
				t.Fatalf("expected error for invalid transition %s -> %s, got nil", tc.fromStatus, tc.toStatus)
			}

			expectedMsg := fmt.Sprintf("invalid transition from '%s' to '%s'", tc.fromStatus, tc.toStatus)
			if err.Error() != expectedMsg {
				t.Errorf("expected error message %q, got %q", expectedMsg, err.Error())
			}

			if err := conn.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet pgxmock expectations: %v", err)
			}
		})
	}
}

func TestUpdateStatus_OrderNotFound(t *testing.T) {
	conn, err := pgxmock.NewConn()
	if err != nil {
		t.Fatalf("failed to create pgxmock conn: %v", err)
	}
	defer conn.Close(context.Background())

	svc := NewOrderService(nil)
	orderID := uuid.New()
	userID := uuid.New()

	conn.ExpectQuery(`SELECT (.+) FROM orders WHERE id = \$1`).
		WithArgs(orderID).
		WillReturnError(fmt.Errorf("no rows in result set"))

	ctx := context.WithValue(context.Background(), middleware.ContextKeyDBTx, conn)

	req := domain.UpdateOrderStatusRequest{
		Status: "washing",
	}

	_, err = svc.UpdateStatus(ctx, orderID, userID, req)
	if err == nil {
		t.Fatal("expected error for order not found, got nil")
	}
}

func TestUpdateStatus_TerminalStatusCannotTransition(t *testing.T) {
	conn, err := pgxmock.NewConn()
	if err != nil {
		t.Fatalf("failed to create pgxmock conn: %v", err)
	}
	defer conn.Close(context.Background())

	orderID := uuid.New()
	tenantID := uuid.New()
	outletID := uuid.New()
	userID := uuid.New()
	now := time.Now()

	orderRows := conn.NewRows([]string{
		"id", "transaction_id", "tenant_id", "outlet_id", "status",
		"estimated_completion_at", "completed_at", "picked_up_at", "notes",
		"customer_phone", "tracking_token",
		"delivery_type", "delivery_address", "delivery_fee", "scheduled_pickup_at",
		"created_by", "updated_by", "created_at", "updated_at",
	}).AddRow(
		orderID, nil, tenantID, outletID, "picked_up",
		nil, nil, &now, nil,
		nil, nil,
		"pickup", nil, int64(0), nil,
		userID, userID, now, now,
	)

	conn.ExpectQuery(`SELECT (.+) FROM orders WHERE id = \$1`).
		WithArgs(orderID).
		WillReturnRows(orderRows)

	svc := NewOrderService(nil)
	ctx := context.WithValue(context.Background(), middleware.ContextKeyDBTx, conn)

	req := domain.UpdateOrderStatusRequest{
		Status: "done",
	}

	_, err = svc.UpdateStatus(ctx, orderID, userID, req)
	if err == nil {
		t.Fatal("expected error for transition from terminal status 'picked_up', got nil")
	}

	expectedMsg := "cannot transition from status 'picked_up'"
	if err.Error() != expectedMsg {
		t.Errorf("expected error message %q, got %q", expectedMsg, err.Error())
	}
}

// --- Pure logic tests for tracking token ---

func TestGenerateTrackingToken_Format(t *testing.T) {
	token := generateTrackingToken()

	if len(token) != 8 {
		t.Errorf("expected tracking token length 8, got %d: %q", len(token), token)
	}

	for _, c := range token {
		if !((c >= '0' && c <= '9') || (c >= 'A' && c <= 'F')) {
			t.Errorf("expected uppercase hex character, got %c in token %q", c, token)
		}
	}
}

func TestGenerateTrackingToken_Uniqueness(t *testing.T) {
	tokens := make(map[string]bool)
	for i := 0; i < 100; i++ {
		token := generateTrackingToken()
		if tokens[token] {
			t.Errorf("duplicate tracking token generated: %s", token)
		}
		tokens[token] = true
	}
}
