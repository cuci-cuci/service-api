#!/bin/bash
# ============================================================
# API Integration Test Suite — LaundryPOS
# Tests all endpoints with real tokens against the live API
# ============================================================

set -euo pipefail

API_BASE="https://api.nirmalab.com/api/v1"
PASS=0
FAIL=0
ERRORS=""

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# --------------------------------------------------------
# Helpers
# --------------------------------------------------------

login() {
  local email="$1" password="$2"
  /usr/bin/curl -s "$API_BASE/auth/login" -X POST \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"$email\",\"password\":\"$password\"}" \
    | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['access_token'])"
}

test_endpoint() {
  local method="$1" path="$2" token="$3" expected_status="$4" label="$5"
  local actual_status body

  if [ "$method" = "GET" ]; then
    actual_status=$(/usr/bin/curl -s -o /tmp/api_test_body -w "%{http_code}" \
      "$API_BASE$path" -H "Authorization: Bearer $token")
  elif [ "$method" = "POST" ]; then
    local data="${6:-{}}"
    actual_status=$(/usr/bin/curl -s -o /tmp/api_test_body -w "%{http_code}" \
      -X POST "$API_BASE$path" \
      -H "Authorization: Bearer $token" \
      -H "Content-Type: application/json" \
      -d "$data")
  elif [ "$method" = "PUT" ]; then
    local data="${6:-{}}"
    actual_status=$(/usr/bin/curl -s -o /tmp/api_test_body -w "%{http_code}" \
      -X PUT "$API_BASE$path" \
      -H "Authorization: Bearer $token" \
      -H "Content-Type: application/json" \
      -d "$data")
  elif [ "$method" = "PATCH" ]; then
    local data="${6:-{}}"
    actual_status=$(/usr/bin/curl -s -o /tmp/api_test_body -w "%{http_code}" \
      -X PATCH "$API_BASE$path" \
      -H "Authorization: Bearer $token" \
      -H "Content-Type: application/json" \
      -d "$data")
  elif [ "$method" = "DELETE" ]; then
    actual_status=$(/usr/bin/curl -s -o /dev/null -w "%{http_code}" \
      -X DELETE "$API_BASE$path" -H "Authorization: Bearer $token")
  fi

  if [ "$actual_status" = "$expected_status" ]; then
    printf "${GREEN}PASS${NC} [%s] %-50s (got %s)\n" "$method" "$label" "$actual_status"
    PASS=$((PASS + 1))
  else
    printf "${RED}FAIL${NC} [%s] %-50s (expected %s, got %s)\n" "$method" "$label" "$expected_status" "$actual_status"
    FAIL=$((FAIL + 1))
    ERRORS="$ERRORS\n  FAIL: [$method] $label — expected $expected_status, got $actual_status"
  fi
}

# --------------------------------------------------------
# Login & get tokens
# --------------------------------------------------------

echo ""
echo "============================================"
echo "  LaundryPOS API Integration Tests"
echo "============================================"
echo ""

echo "Authenticating..."
ADMIN_TOKEN=$(login "admin@laundry.app" "admin123456")
OWNER_TOKEN=$(login "owner@demo.cuci.app" "owner123456")
CASHIER_TOKEN=$(login "kasir@demo.cuci.app" "kasir123456")

echo "  Superadmin: ${ADMIN_TOKEN:0:20}..."
echo "  Owner:      ${OWNER_TOKEN:0:20}..."
echo "  Cashier:    ${CASHIER_TOKEN:0:20}..."
echo ""

# --------------------------------------------------------
# 1. PUBLIC ENDPOINTS
# --------------------------------------------------------

echo "--- PUBLIC ---"
test_endpoint GET "/auth/login" "" "405" "Login requires POST"
test_endpoint GET "/plans" "" "200" "List subscription plans"

# --------------------------------------------------------
# 2. ADMIN ENDPOINTS
# --------------------------------------------------------

echo ""
echo "--- ADMIN (superadmin) ---"
test_endpoint GET "/admin/dashboard/stats" "$ADMIN_TOKEN" "200" "Dashboard stats"
test_endpoint GET "/admin/dashboard/revenue" "$ADMIN_TOKEN" "200" "Dashboard revenue"
test_endpoint GET "/admin/dashboard/tenant-health" "$ADMIN_TOKEN" "200" "Dashboard tenant health"
test_endpoint GET "/admin/tenants" "$ADMIN_TOKEN" "200" "List tenants"
test_endpoint GET "/admin/users" "$ADMIN_TOKEN" "200" "List users"
test_endpoint GET "/admin/service-categories" "$ADMIN_TOKEN" "200" "List service categories"
test_endpoint GET "/admin/service-templates" "$ADMIN_TOKEN" "200" "List service templates"
test_endpoint GET "/admin/feature-flags" "$ADMIN_TOKEN" "200" "List feature flags"
test_endpoint GET "/admin/membership" "$ADMIN_TOKEN" "200" "List memberships"
test_endpoint GET "/admin/audit-logs" "$ADMIN_TOKEN" "200" "List audit logs"
test_endpoint GET "/admin/transactions" "$ADMIN_TOKEN" "200" "List transactions"
test_endpoint GET "/admin/sync/health/outlets" "$ADMIN_TOKEN" "200" "Sync health outlets"
test_endpoint GET "/admin/sync/sessions" "$ADMIN_TOKEN" "200" "Sync sessions"
test_endpoint GET "/admin/configs" "$ADMIN_TOKEN" "200" "List configs"
test_endpoint GET "/admin/inventory/summary" "$ADMIN_TOKEN" "200" "Inventory summary"
test_endpoint GET "/admin/inventory/alerts" "$ADMIN_TOKEN" "200" "Inventory alerts"
test_endpoint GET "/admin/inventory/supplies?tenant_id=e701d2d3-f5d6-45d2-8202-fb514a695377" "$ADMIN_TOKEN" "200" "Inventory supplies"
test_endpoint GET "/admin/orders/summary" "$ADMIN_TOKEN" "200" "Orders summary"
test_endpoint GET "/admin/orders" "$ADMIN_TOKEN" "200" "List orders"

# Admin with wrong role
echo ""
echo "--- ADMIN (access control) ---"
test_endpoint GET "/admin/tenants" "$OWNER_TOKEN" "403" "Owner blocked from admin"
test_endpoint GET "/admin/tenants" "$CASHIER_TOKEN" "403" "Cashier blocked from admin"

# --------------------------------------------------------
# 3. OWNER ENDPOINTS
# --------------------------------------------------------

echo ""
echo "--- OWNER: Core ---"
test_endpoint GET "/owner/outlets" "$OWNER_TOKEN" "200" "List outlets"
test_endpoint GET "/owner/services" "$OWNER_TOKEN" "200" "List services"
test_endpoint GET "/owner/payment-methods" "$OWNER_TOKEN" "200" "List payment methods"
test_endpoint GET "/owner/cashiers" "$OWNER_TOKEN" "200" "List cashiers"
test_endpoint GET "/owner/members" "$OWNER_TOKEN" "200" "List members"
test_endpoint GET "/owner/store-settings" "$OWNER_TOKEN" "200" "Store settings"
test_endpoint GET "/owner/subscription" "$OWNER_TOKEN" "200" "Subscription"
test_endpoint GET "/owner/notification-settings" "$OWNER_TOKEN" "200" "Notification settings"

echo ""
echo "--- OWNER: Dashboard ---"
test_endpoint GET "/owner/dashboard/summary" "$OWNER_TOKEN" "200" "Dashboard summary"
test_endpoint GET "/owner/dashboard/cashier-performance" "$OWNER_TOKEN" "200" "Cashier performance"
test_endpoint GET "/owner/dashboard/customer-insights" "$OWNER_TOKEN" "200" "Customer insights"
test_endpoint GET "/owner/dashboard/goals" "$OWNER_TOKEN" "200" "Dashboard goals"
test_endpoint GET "/owner/dashboard/summary/range?start=2026-03-01&end=2026-03-15" "$OWNER_TOKEN" "200" "Dashboard range"

echo ""
echo "--- OWNER: Analytics ---"
test_endpoint GET "/owner/analytics/summary" "$OWNER_TOKEN" "200" "Analytics summary"
test_endpoint GET "/owner/analytics/daily-revenue" "$OWNER_TOKEN" "200" "Daily revenue"
test_endpoint GET "/owner/analytics/by-service" "$OWNER_TOKEN" "200" "Revenue by service"
test_endpoint GET "/owner/analytics/by-payment-method" "$OWNER_TOKEN" "200" "Revenue by payment method"
test_endpoint GET "/owner/analytics/outlets" "$OWNER_TOKEN" "200" "Revenue by outlet"

echo ""
echo "--- OWNER: Finance ---"
test_endpoint GET "/owner/expense-categories" "$OWNER_TOKEN" "200" "Expense categories"
test_endpoint GET "/owner/expenses" "$OWNER_TOKEN" "200" "List expenses"
test_endpoint GET "/owner/recurring-expenses" "$OWNER_TOKEN" "200" "Recurring expenses"
test_endpoint GET "/owner/finance/pnl?start_date=2026-02-01&end_date=2026-03-16" "$OWNER_TOKEN" "200" "P&L report"
test_endpoint GET "/owner/finance/cashflow?start_date=2026-02-01&end_date=2026-03-16" "$OWNER_TOKEN" "200" "Cash flow report"
test_endpoint GET "/owner/finance/tax?start_date=2026-02-01&end_date=2026-03-16" "$OWNER_TOKEN" "200" "Tax report"

echo ""
echo "--- OWNER: Inventory ---"
test_endpoint GET "/owner/supply-categories" "$OWNER_TOKEN" "200" "Supply categories"
test_endpoint GET "/owner/supplies" "$OWNER_TOKEN" "200" "List supplies"
test_endpoint GET "/owner/stock-movements" "$OWNER_TOKEN" "200" "Stock movements"
test_endpoint GET "/owner/stock-alerts" "$OWNER_TOKEN" "200" "Stock alerts"
test_endpoint GET "/owner/service-supply-mappings" "$OWNER_TOKEN" "200" "Service-supply mappings"
test_endpoint GET "/owner/service-costs" "$OWNER_TOKEN" "200" "Service costs"

echo ""
echo "--- OWNER: Delivery ---"
test_endpoint GET "/owner/delivery-zones" "$OWNER_TOKEN" "200" "Delivery zones"
test_endpoint GET "/owner/pickup-requests" "$OWNER_TOKEN" "200" "Pickup requests"

echo ""
echo "--- OWNER: Staff ---"
test_endpoint GET "/owner/staff/activities" "$OWNER_TOKEN" "200" "Staff activities"
test_endpoint GET "/owner/staff/summaries" "$OWNER_TOKEN" "200" "Staff summaries"

echo ""
echo "--- OWNER: Gateway ---"
test_endpoint GET "/owner/gateway-config" "$OWNER_TOKEN" "200" "Gateway config"
test_endpoint GET "/owner/gateway-payments" "$OWNER_TOKEN" "200" "Gateway payments"

# Owner access control
echo ""
echo "--- OWNER (access control) ---"
test_endpoint GET "/owner/outlets" "$CASHIER_TOKEN" "403" "Cashier blocked from owner"

# --------------------------------------------------------
# 4. POS ENDPOINTS
# --------------------------------------------------------

echo ""
echo "--- POS (cashier) ---"
test_endpoint GET "/pos/shifts/current" "$CASHIER_TOKEN" "200" "Current shift"
test_endpoint GET "/pos/shifts" "$CASHIER_TOKEN" "200" "List shifts"
test_endpoint GET "/pos/orders/active" "$CASHIER_TOKEN" "200" "Active orders"
test_endpoint GET "/pos/orders" "$CASHIER_TOKEN" "200" "List orders"
test_endpoint GET "/pos/outlets" "$CASHIER_TOKEN" "200" "List outlets"
test_endpoint GET "/pos/transactions" "$CASHIER_TOKEN" "200" "List transactions"
test_endpoint GET "/pos/delivery/zones" "$CASHIER_TOKEN" "200" "Delivery zones (POS)"

# --------------------------------------------------------
# 5. DATA INTEGRITY CHECKS
# --------------------------------------------------------

echo ""
echo "--- DATA INTEGRITY ---"

# Check transactions have data
TXNS=$(/usr/bin/curl -s "$API_BASE/owner/analytics/summary" -H "Authorization: Bearer $OWNER_TOKEN" | python3 -c "import sys,json; d=json.load(sys.stdin).get('data',{}); print(d.get('total_transactions',0))" 2>/dev/null || echo "0")
if [ "$TXNS" -gt "0" ] 2>/dev/null; then
  printf "${GREEN}PASS${NC} %-58s (count: %s)\n" "Transactions exist in analytics" "$TXNS"
  PASS=$((PASS + 1))
else
  printf "${RED}FAIL${NC} %-58s (count: %s)\n" "Transactions exist in analytics" "$TXNS"
  FAIL=$((FAIL + 1))
  ERRORS="$ERRORS\n  FAIL: No transactions found in analytics summary"
fi

# Check members have data
MEMBERS=$(/usr/bin/curl -s "$API_BASE/owner/members" -H "Authorization: Bearer $OWNER_TOKEN" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('meta',{}).get('total',0))" 2>/dev/null || echo "0")
if [ "$MEMBERS" -gt "5" ] 2>/dev/null; then
  printf "${GREEN}PASS${NC} %-58s (count: %s)\n" "Members seeded (expected 10)" "$MEMBERS"
  PASS=$((PASS + 1))
else
  printf "${RED}FAIL${NC} %-58s (count: %s)\n" "Members seeded (expected 10)" "$MEMBERS"
  FAIL=$((FAIL + 1))
  ERRORS="$ERRORS\n  FAIL: Expected >5 members, got $MEMBERS"
fi

# Check expenses exist
EXPENSES=$(/usr/bin/curl -s "$API_BASE/owner/expenses" -H "Authorization: Bearer $OWNER_TOKEN" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('meta',{}).get('total', len(d.get('data',[]))))" 2>/dev/null || echo "0")
if [ "$EXPENSES" -gt "0" ] 2>/dev/null; then
  printf "${GREEN}PASS${NC} %-58s (count: %s)\n" "Expenses seeded" "$EXPENSES"
  PASS=$((PASS + 1))
else
  printf "${RED}FAIL${NC} %-58s (count: %s)\n" "Expenses seeded" "$EXPENSES"
  FAIL=$((FAIL + 1))
  ERRORS="$ERRORS\n  FAIL: No expenses found"
fi

# Check delivery zones exist
ZONES=$(/usr/bin/curl -s "$API_BASE/owner/delivery-zones" -H "Authorization: Bearer $OWNER_TOKEN" | python3 -c "import sys,json; d=json.load(sys.stdin); print(len(d.get('data',[])))" 2>/dev/null || echo "0")
if [ "$ZONES" -gt "0" ] 2>/dev/null; then
  printf "${GREEN}PASS${NC} %-58s (count: %s)\n" "Delivery zones seeded" "$ZONES"
  PASS=$((PASS + 1))
else
  printf "${RED}FAIL${NC} %-58s (count: %s)\n" "Delivery zones seeded" "$ZONES"
  FAIL=$((FAIL + 1))
  ERRORS="$ERRORS\n  FAIL: No delivery zones found"
fi

# Check supplies exist
SUPPLIES=$(/usr/bin/curl -s "$API_BASE/owner/supplies" -H "Authorization: Bearer $OWNER_TOKEN" | python3 -c "import sys,json; d=json.load(sys.stdin); print(len(d.get('data',[])))" 2>/dev/null || echo "0")
if [ "$SUPPLIES" -gt "0" ] 2>/dev/null; then
  printf "${GREEN}PASS${NC} %-58s (count: %s)\n" "Supplies seeded" "$SUPPLIES"
  PASS=$((PASS + 1))
else
  printf "${RED}FAIL${NC} %-58s (count: %s)\n" "Supplies seeded" "$SUPPLIES"
  FAIL=$((FAIL + 1))
  ERRORS="$ERRORS\n  FAIL: No supplies found"
fi

# --------------------------------------------------------
# SUMMARY
# --------------------------------------------------------

echo ""
echo "============================================"
TOTAL=$((PASS + FAIL))
echo "  Results: $PASS/$TOTAL passed"
if [ $FAIL -gt 0 ]; then
  printf "  ${RED}$FAIL FAILED${NC}\n"
  printf "\n  Failed tests:$ERRORS\n"
  echo ""
  echo "============================================"
  exit 1
else
  printf "  ${GREEN}ALL TESTS PASSED${NC}\n"
  echo "============================================"
  exit 0
fi
