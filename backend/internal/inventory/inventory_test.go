package inventory

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"inventarius/internal/db"
)

type fixture struct {
	ctx        context.Context
	svc        *Service
	q          *db.Store
	user       db.User
	item       db.Item
	shelfA     db.Location
	shelfB     db.Location
	propulsion int64 // department id
}

func setup(t *testing.T) *fixture {
	t.Helper()
	ctx := context.Background()
	store, err := db.Open(ctx, ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	f := &fixture{ctx: ctx, svc: &Service{Store: store}, q: store}
	if f.user, err = store.CreateUser(ctx, db.CreateUserParams{Email: "a@portospaceteam.pt", Name: "Ana", Role: "member"}); err != nil {
		t.Fatal(err)
	}
	if f.item, err = store.CreateItem(ctx, db.CreateItemParams{Name: "M3 screw", Kind: "consumable", Unit: "pcs"}); err != nil {
		t.Fatal(err)
	}
	f.shelfA, _ = store.CreateLocation(ctx, db.CreateLocationParams{Name: "Shelf A"})
	f.shelfB, _ = store.CreateLocation(ctx, db.CreateLocationParams{Name: "Shelf B"})
	deps, _ := store.ListDepartments(ctx)
	for _, d := range deps {
		if d.Name == "Propulsion" {
			f.propulsion = d.ID
		}
	}
	return f
}

func (f *fixture) receive(t *testing.T, loc int64, qty float64) db.Stock {
	t.Helper()
	s, err := f.svc.Receive(f.ctx, f.user.ID, ReceiveParams{ItemID: f.item.ID, LocationID: loc, Quantity: qty})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func (f *fixture) qty(t *testing.T, id int64) float64 {
	t.Helper()
	s, err := f.q.GetStock(f.ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	return s.Quantity
}

func (f *fixture) movements(t *testing.T) []db.ListMovementsRow {
	t.Helper()
	m, err := f.q.ListMovements(f.ctx, db.ListMovementsParams{Lim: 100})
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func isUserErr(err error) bool {
	var ue *UserError
	return errors.As(err, &ue)
}

func TestReceiveMergesSameLot(t *testing.T) {
	f := setup(t)
	a := f.receive(t, f.shelfA.ID, 10)
	b := f.receive(t, f.shelfA.ID, 5)
	if a.ID != b.ID || f.qty(t, a.ID) != 15 {
		t.Fatalf("expected one lot of 15, got %d/%d qty %v", a.ID, b.ID, f.qty(t, a.ID))
	}
	c, _ := f.svc.Receive(f.ctx, f.user.ID, ReceiveParams{ItemID: f.item.ID, LocationID: f.shelfA.ID, Quantity: 1, Lot: "2026-01"})
	if c.ID == a.ID {
		t.Fatal("different lot merged")
	}
	if n := len(f.movements(t)); n != 3 {
		t.Fatalf("%d movements, want 3", n)
	}
}

func TestTakeAndReturn(t *testing.T) {
	f := setup(t)
	s := f.receive(t, f.shelfA.ID, 10)
	dep := f.propulsion
	if _, err := f.svc.Take(f.ctx, f.user.ID, s.ID, 3, Purpose{DepartmentID: &dep}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Take(f.ctx, f.user.ID, s.ID, 8, Purpose{}); !isUserErr(err) {
		t.Fatalf("taking more than available: %v", err)
	}
	if _, err := f.svc.Take(f.ctx, f.user.ID, s.ID, -1, Purpose{}); !isUserErr(err) {
		t.Fatalf("negative take: %v", err)
	}
	if _, err := f.svc.Return(f.ctx, f.user.ID, s.ID, 0.5, Purpose{}); err != nil {
		t.Fatal(err)
	}
	if q := f.qty(t, s.ID); q != 7.5 {
		t.Fatalf("qty %v, want 7.5", q)
	}
	m := f.movements(t)[1] // newest first: return, take, receive
	if m.Movement.Kind != "take" || m.Movement.Quantity != -3 || m.Movement.ProjectID == nil || *m.Movement.ProjectID != 1 {
		t.Fatalf("take movement %+v: department should imply project INVICTUS", m.Movement)
	}
}

func TestTakeRejectsDepartmentFromOtherProject(t *testing.T) {
	f := setup(t)
	s := f.receive(t, f.shelfA.ID, 10)
	icarus, dep := int64(2), f.propulsion
	if _, err := f.svc.Take(f.ctx, f.user.ID, s.ID, 1, Purpose{ProjectID: &icarus, DepartmentID: &dep}); !isUserErr(err) {
		t.Fatalf("expected user error, got %v", err)
	}
}

func TestMove(t *testing.T) {
	f := setup(t)
	s := f.receive(t, f.shelfA.ID, 10)

	part, err := f.svc.Move(f.ctx, f.user.ID, s.ID, f.shelfB.ID, 4, "")
	if err != nil {
		t.Fatal(err)
	}
	if part.ID == s.ID || f.qty(t, s.ID) != 6 || f.qty(t, part.ID) != 4 {
		t.Fatalf("partial move: src %v dest %v", f.qty(t, s.ID), f.qty(t, part.ID))
	}
	// Moving the rest merges into the lot already on shelf B.
	merged, err := f.svc.Move(f.ctx, f.user.ID, s.ID, f.shelfB.ID, 6, "")
	if err != nil {
		t.Fatal(err)
	}
	if merged.ID != part.ID || f.qty(t, part.ID) != 10 || f.qty(t, s.ID) != 0 {
		t.Fatalf("merge: dest %d qty %v src qty %v", merged.ID, f.qty(t, part.ID), f.qty(t, s.ID))
	}
	// Moving back merges into the emptied (quantity 0) lot still on shelf A.
	back, err := f.svc.Move(f.ctx, f.user.ID, part.ID, f.shelfA.ID, 10, "")
	if err != nil || back.ID != s.ID || f.qty(t, s.ID) != 10 {
		t.Fatalf("move back: %v, dest %d", err, back.ID)
	}
	if _, err := f.svc.Move(f.ctx, f.user.ID, s.ID, f.shelfA.ID, 1, ""); !isUserErr(err) {
		t.Fatalf("move to same location: %v", err)
	}
}

func TestMoveWholeLotKeepsRow(t *testing.T) {
	f := setup(t)
	s := f.receive(t, f.shelfA.ID, 3)
	got, err := f.svc.Move(f.ctx, f.user.ID, s.ID, f.shelfB.ID, 3, "")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != s.ID || got.LocationID != f.shelfB.ID {
		t.Fatalf("whole move should relocate the row: %+v", got)
	}
}

func TestAdjust(t *testing.T) {
	f := setup(t)
	s := f.receive(t, f.shelfA.ID, 10)
	if _, err := f.svc.Adjust(f.ctx, f.user.ID, s.ID, 7, "", "stocktake"); err != nil {
		t.Fatal(err)
	}
	if m := f.movements(t)[0].Movement; m.Kind != "adjust" || m.Quantity != -3 {
		t.Fatalf("adjust movement %+v", m)
	}
	before := len(f.movements(t))
	f.svc.Adjust(f.ctx, f.user.ID, s.ID, 7, "", "")
	if len(f.movements(t)) != before {
		t.Fatal("no-op adjust logged a movement")
	}
}

func TestBorrowAndGiveBack(t *testing.T) {
	f := setup(t)
	scope, _ := f.q.CreateItem(f.ctx, db.CreateItemParams{Name: "Oscilloscope", Kind: "tool", Unit: "pcs"})
	a, err := f.svc.CreateAsset(f.ctx, f.user.ID, AssetParams{ItemID: scope.ID, LocationID: &f.shelfA.ID})
	if err != nil {
		t.Fatal(err)
	}
	other, _ := f.q.CreateUser(f.ctx, db.CreateUserParams{Email: "b@portospaceteam.pt", Name: "Bruno", Role: "member"})

	if _, err := f.svc.Borrow(f.ctx, f.user.ID, BorrowParams{AssetID: a.ID}); err != nil {
		t.Fatal(err)
	}
	_, err = f.svc.Borrow(f.ctx, other.ID, BorrowParams{AssetID: a.ID})
	if !isUserErr(err) || err.Error() != "Ana already has it" {
		t.Fatalf("second borrow: %v", err)
	}
	if _, err := f.svc.UpdateAsset(f.ctx, f.user.ID, a.ID, AssetParams{Status: "broken"}); !isUserErr(err) {
		t.Fatalf("status change while on loan: %v", err)
	}
	// Someone else returns it to shelf B.
	if err := f.svc.GiveBack(f.ctx, other.ID, a.ID, &f.shelfB.ID, ""); err != nil {
		t.Fatal(err)
	}
	got, _ := f.q.GetAsset(f.ctx, a.ID)
	if got.Status != "available" || *got.LocationID != f.shelfB.ID {
		t.Fatalf("after return: %+v", got)
	}
	if err := f.svc.GiveBack(f.ctx, other.ID, a.ID, nil, ""); !isUserErr(err) {
		t.Fatalf("double return: %v", err)
	}
	loans, _ := f.q.ListOpenLoans(f.ctx, nil)
	if len(loans) != 0 {
		t.Fatalf("%d open loans", len(loans))
	}
}

func TestNotFound(t *testing.T) {
	f := setup(t)
	if _, err := f.svc.Take(f.ctx, f.user.ID, 999, 1, Purpose{}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("got %v", err)
	}
}

func isConflict(err error) bool {
	var ce *ConflictError
	return errors.As(err, &ce)
}

func TestReservedStock(t *testing.T) {
	f := setup(t)
	s := f.receive(t, f.shelfA.ID, 100)
	dep := f.propulsion
	res, err := f.svc.Reserve(f.ctx, f.user.ID, ReserveParams{
		StockID: &s.ID, Quantity: 20, Purpose: Purpose{DepartmentID: &dep, Note: "engine test 3"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Reserve(f.ctx, f.user.ID, ReserveParams{StockID: &s.ID, Quantity: 81, Purpose: Purpose{Note: "x"}}); !isUserErr(err) {
		t.Fatalf("over-reserving: %v", err)
	}
	if _, err := f.svc.Reserve(f.ctx, f.user.ID, ReserveParams{StockID: &s.ID, Quantity: 1}); !isUserErr(err) {
		t.Fatalf("reservation without a note: %v", err)
	}

	// 80 are free; taking more asks for confirmation.
	_, err = f.svc.Take(f.ctx, f.user.ID, s.ID, 85, Purpose{})
	if !isConflict(err) || !strings.Contains(err.Error(), "INVICTUS · Propulsion") {
		t.Fatalf("taking reserved stock: %v", err)
	}
	if _, err := f.svc.Take(f.ctx, f.user.ID, s.ID, 80, Purpose{}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Take(f.ctx, f.user.ID, s.ID, 5, Purpose{OverrideReserved: true}); err != nil {
		t.Fatal(err)
	}
	if m := f.movements(t)[0].Movement; !strings.Contains(m.Note, "used reserved stock") {
		t.Fatalf("override not recorded: %+v", m)
	}

	// Taking from the reservation itself uses it up, for its purpose.
	if _, err := f.svc.Take(f.ctx, f.user.ID, s.ID, 15, Purpose{ReservationID: &res.ID}); err != nil {
		t.Fatal(err)
	}
	m := f.movements(t)[0].Movement
	if m.DepartmentID == nil || *m.DepartmentID != dep || m.ReservationID == nil {
		t.Fatalf("take from reservation: %+v", m)
	}
	r, _ := f.q.GetReservation(f.ctx, res.ID)
	if r.Quantity != 5 || r.ReleasedAt != nil {
		t.Fatalf("reservation after use: %+v", r)
	}
	// The override took 5 of the reserved parts, so the shelf is empty while
	// 5 are still reserved. Leftovers come back, then the reservation is used up.
	if _, err := f.svc.Return(f.ctx, f.user.ID, s.ID, 5, Purpose{}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Take(f.ctx, f.user.ID, s.ID, 5, Purpose{ReservationID: &res.ID}); err != nil {
		t.Fatal(err)
	}
	if r, _ := f.q.GetReservation(f.ctx, res.ID); r.ReleasedAt == nil {
		t.Fatal("used-up reservation still active")
	}
}

func TestExpiredReservationIsFree(t *testing.T) {
	f := setup(t)
	s := f.receive(t, f.shelfA.ID, 10)
	past := time.Now().UTC().AddDate(0, 0, -2)
	if _, err := f.q.CreateReservation(f.ctx, db.CreateReservationParams{
		ItemID: s.ItemID, StockID: &s.ID, Quantity: 10, UserID: f.user.ID, Note: "old", Until: &past,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Take(f.ctx, f.user.ID, s.ID, 10, Purpose{}); err != nil {
		t.Fatalf("expired reservation still blocks: %v", err)
	}
	today := time.Now().UTC()
	if _, err := f.svc.Reserve(f.ctx, f.user.ID, ReserveParams{StockID: &s.ID, Quantity: 1, Purpose: Purpose{Note: "x"}, Until: &today}); !isUserErr(err) {
		t.Fatalf("reserving from empty stock: %v", err)
	}
}

func TestReservedAsset(t *testing.T) {
	f := setup(t)
	tool, _ := f.q.CreateItem(f.ctx, db.CreateItemParams{Name: "Torque wrench", Kind: "tool", Unit: "pcs"})
	a, _ := f.svc.CreateAsset(f.ctx, f.user.ID, AssetParams{ItemID: tool.ID})
	dep := f.propulsion
	res, err := f.svc.Reserve(f.ctx, f.user.ID, ReserveParams{AssetID: &a.ID, Purpose: Purpose{DepartmentID: &dep, Note: "assembly day"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Reserve(f.ctx, f.user.ID, ReserveParams{AssetID: &a.ID, Purpose: Purpose{Note: "x"}}); !isUserErr(err) {
		t.Fatalf("double reservation: %v", err)
	}
	other, _ := f.q.CreateUser(f.ctx, db.CreateUserParams{Email: "c@portospaceteam.pt", Name: "Carla", Role: "member"})
	if _, err := f.svc.Borrow(f.ctx, other.ID, BorrowParams{AssetID: a.ID}); !isConflict(err) {
		t.Fatalf("borrowing someone else's reserved tool: %v", err)
	}
	// Same department: fine.
	l, err := f.svc.Borrow(f.ctx, other.ID, BorrowParams{AssetID: a.ID, Purpose: Purpose{DepartmentID: &dep}})
	if err != nil {
		t.Fatal(err)
	}
	f.svc.GiveBack(f.ctx, other.ID, l.AssetID, nil, "")
	if _, err := f.svc.Borrow(f.ctx, other.ID, BorrowParams{AssetID: a.ID, Purpose: Purpose{OverrideReserved: true}}); err != nil {
		t.Fatal(err)
	}
	if err := f.svc.Release(f.ctx, f.user.ID, res.ID); err != nil {
		t.Fatal(err)
	}
	if err := f.svc.Release(f.ctx, f.user.ID, res.ID); !isUserErr(err) {
		t.Fatalf("double release: %v", err)
	}
	if m := f.movements(t)[0].Movement; m.Kind != "unreserve" {
		t.Fatalf("release not logged: %+v", m)
	}
}
