package inventory

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"inventarius/internal/db"
)

// ConflictError means the request would use something reserved for someone
// else. The caller may repeat it with Override after the user confirms.
type ConflictError struct{ Msg string }

func (e *ConflictError) Error() string { return e.Msg }

type ReserveParams struct {
	StockID  *int64
	AssetID  *int64
	Quantity float64 // ignored for assets
	Purpose  Purpose // who it is for; Note says what for (required)
	Until    *time.Time
}

// Reserve sets aside part of a stock lot, or an asset, for a purpose.
func (s *Service) Reserve(ctx context.Context, userID int64, p ReserveParams) (db.Reservation, error) {
	if (p.StockID == nil) == (p.AssetID == nil) {
		return db.Reservation{}, userErr("reserve either stock or a unit")
	}
	p.Purpose.Note = strings.TrimSpace(p.Purpose.Note)
	if p.Purpose.Note == "" {
		return db.Reservation{}, userErr("say what it is reserved for")
	}
	if p.Until != nil {
		d := p.Until.UTC().Truncate(24 * time.Hour)
		if d.Before(time.Now().UTC().Truncate(24 * time.Hour)) {
			return db.Reservation{}, userErr("the end date is in the past")
		}
		p.Until = &d
	}
	var res db.Reservation
	err := s.Store.Tx(ctx, func(q *db.Queries) error {
		why, err := purposeProject(ctx, q, p.Purpose)
		if err != nil {
			return err
		}
		params := db.CreateReservationParams{
			StockID: p.StockID, AssetID: p.AssetID, UserID: userID,
			ProjectID: why.ProjectID, DepartmentID: why.DepartmentID, Note: why.Note, Until: p.Until,
		}
		m := db.CreateMovementParams{UserID: userID, Kind: "reserve", ProjectID: why.ProjectID, DepartmentID: why.DepartmentID, Note: why.Note}
		if p.StockID != nil {
			qty, err := positive(p.Quantity)
			if err != nil {
				return err
			}
			st, err := q.GetStock(ctx, *p.StockID)
			if err != nil {
				return notFound(err)
			}
			reserved, err := q.ReservedOnStock(ctx, &st.ID)
			if err != nil {
				return err
			}
			if free := round(st.Quantity - reserved); st.CountState != Uncounted && qty > free {
				return userErr("only %g free to reserve here", max(free, 0))
			}
			params.ItemID, params.Quantity = st.ItemID, qty
			m.ItemID, m.StockID, m.Quantity = st.ItemID, &st.ID, qty
		} else {
			a, err := q.GetAsset(ctx, *p.AssetID)
			if err != nil {
				return notFound(err)
			}
			if a.Status == "retired" {
				return userErr("it is retired")
			}
			if other, err := q.ActiveReservationForAsset(ctx, &a.ID); err == nil {
				return userErr("it is already reserved for %s", purposeLabel(ctx, q, other.ProjectID, other.DepartmentID))
			} else if !errors.Is(err, sql.ErrNoRows) {
				return err
			}
			params.ItemID, params.Quantity = a.ItemID, 1
			m.ItemID, m.AssetID, m.Quantity = a.ItemID, &a.ID, 1
		}
		if res, err = q.CreateReservation(ctx, params); err != nil {
			return err
		}
		m.ReservationID = &res.ID
		return q.CreateMovement(ctx, m)
	})
	return res, err
}

// Release ends a reservation early.
func (s *Service) Release(ctx context.Context, userID, reservationID int64) error {
	return s.Store.Tx(ctx, func(q *db.Queries) error {
		r, err := q.GetReservation(ctx, reservationID)
		if err != nil {
			return notFound(err)
		}
		if r.ReleasedAt != nil {
			return userErr("it was already released")
		}
		if err := q.ReleaseReservation(ctx, r.ID); err != nil {
			return err
		}
		return q.CreateMovement(ctx, db.CreateMovementParams{
			UserID: userID, Kind: "unreserve", ItemID: r.ItemID, StockID: r.StockID, AssetID: r.AssetID,
			Quantity: r.Quantity, ProjectID: r.ProjectID, DepartmentID: r.DepartmentID, Note: r.Note,
			ReservationID: &r.ID,
		})
	})
}

func active(r db.Reservation) bool {
	return r.ReleasedAt == nil && (r.Until == nil || !r.Until.Before(time.Now().UTC().Truncate(24*time.Hour)))
}

// takeFromStock decides how much of a take or write-off comes out of a
// reservation, and refuses to dip into other reservations unless overridden.
// It returns the reservation used (if any) and a note for the movement.
func takeFromStock(ctx context.Context, q *db.Queries, st db.Stock, qty float64, why *Purpose) (*int64, string, error) {
	reserved, err := q.ReservedOnStock(ctx, &st.ID)
	if err != nil {
		return nil, "", err
	}
	free := round(st.Quantity - reserved)
	fromFree := qty
	var used *int64

	if why.ReservationID != nil {
		r, err := q.GetReservation(ctx, *why.ReservationID)
		if err != nil || r.StockID == nil || *r.StockID != st.ID || !active(r) {
			return nil, "", userErr("that reservation is not active for this stock")
		}
		use := min(qty, r.Quantity)
		fromFree = round(qty - use)
		left := round(r.Quantity - use)
		if left == 0 {
			err = q.ReleaseReservation(ctx, r.ID)
		} else {
			err = q.SetReservationQuantity(ctx, db.SetReservationQuantityParams{Quantity: left, ID: r.ID})
		}
		if err != nil {
			return nil, "", err
		}
		if why.ProjectID == nil && why.DepartmentID == nil {
			why.ProjectID, why.DepartmentID = r.ProjectID, r.DepartmentID
		}
		used = &r.ID
	}

	// Without a count there is no "free" amount to protect.
	if fromFree <= max(free, 0) || st.CountState == Uncounted {
		return used, "", nil
	}
	if !why.OverrideReserved {
		return nil, "", &ConflictError{fmt.Sprintf("Only %g free here; the rest is reserved: %s.",
			max(free, 0), reservationSummary(ctx, q, st))}
	}
	return used, "used reserved stock", nil
}

func reservationSummary(ctx context.Context, q *db.Queries, st db.Stock) string {
	rows, err := q.ListActiveReservationsForItem(ctx, st.ItemID)
	if err != nil {
		return "unknown"
	}
	var parts []string
	for _, r := range rows {
		if r.Reservation.StockID != nil && *r.Reservation.StockID == st.ID {
			parts = append(parts, fmt.Sprintf("%g for %s (%q, %s)", r.Reservation.Quantity,
				purposeLabel(ctx, q, r.Reservation.ProjectID, r.Reservation.DepartmentID), r.Reservation.Note, r.UserName))
		}
	}
	return strings.Join(parts, "; ")
}

// borrowCheck refuses to lend an asset reserved for someone else unless
// overridden. Borrowing it for the reserving department/project, or by the
// person who reserved it, is fine.
func borrowCheck(ctx context.Context, q *db.Queries, userID int64, a db.Asset, why Purpose) (string, error) {
	r, err := q.ActiveReservationForAsset(ctx, &a.ID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	} else if err != nil {
		return "", err
	}
	sameDept := r.DepartmentID != nil && why.DepartmentID != nil && *r.DepartmentID == *why.DepartmentID
	sameProject := r.DepartmentID == nil && r.ProjectID != nil && why.ProjectID != nil && *r.ProjectID == *why.ProjectID
	if r.UserID == userID || sameDept || sameProject {
		return "", nil
	}
	if why.OverrideReserved {
		return "borrowed while reserved for " + purposeLabel(ctx, q, r.ProjectID, r.DepartmentID), nil
	}
	return "", &ConflictError{fmt.Sprintf("It is reserved for %s (%q).",
		purposeLabel(ctx, q, r.ProjectID, r.DepartmentID), r.Note)}
}

// purposeLabel names a project/department for messages, e.g. "INVICTUS · Propulsion".
func purposeLabel(ctx context.Context, q *db.Queries, projectID, departmentID *int64) string {
	projects, _ := q.ListProjects(ctx)
	projectName := func(id int64) string {
		for _, p := range projects {
			if p.ID == id {
				return p.Code
			}
		}
		return "?"
	}
	if departmentID != nil {
		if d, err := q.GetDepartment(ctx, *departmentID); err == nil {
			return projectName(d.ProjectID) + " · " + d.Name
		}
	}
	if projectID != nil {
		return projectName(*projectID)
	}
	return "the team"
}

func joinNotes(a, b string) string {
	switch {
	case a == "":
		return b
	case b == "":
		return a
	}
	return a + " (" + b + ")"
}
