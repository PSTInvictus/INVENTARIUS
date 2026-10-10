package api

import (
	"context"
	"net/http"

	"inventarius/internal/auth"
	"inventarius/internal/db"
	"inventarius/internal/inventory"
)

type reservationView struct {
	db.Reservation
	UserName string `json:"user_name"`
}

// reservations lists an item's active reservations, optionally filtered.
func (s *Server) reservations(ctx context.Context, itemID int64, keep func(db.Reservation) bool) ([]reservationView, error) {
	rows, err := s.store.ListActiveReservationsForItem(ctx, itemID)
	if err != nil {
		return nil, err
	}
	out := []reservationView{}
	for _, r := range rows {
		if keep == nil || keep(r.Reservation) {
			out = append(out, reservationView{Reservation: r.Reservation, UserName: r.UserName})
		}
	}
	return out, nil
}

func (s *Server) reserve(w http.ResponseWriter, r *http.Request, u *auth.User) error {
	var b struct {
		StockID  *int64  `json:"stock_id"`
		AssetID  *int64  `json:"asset_id"`
		Quantity float64 `json:"quantity"`
		Until    date    `json:"until"`
		purposeBody
	}
	if err := decode(r, &b); err != nil {
		return err
	}
	res, err := s.inv.Reserve(r.Context(), u.ID, inventory.ReserveParams{
		StockID: b.StockID, AssetID: b.AssetID, Quantity: b.Quantity, Until: b.Until.t,
		Purpose: inventory.Purpose{ProjectID: b.ProjectID, DepartmentID: b.DepartmentID, Note: b.Note},
	})
	if err != nil {
		return err
	}
	return writeJSON(w, res)
}

// release: the person who reserved it, its project/department, or an admin.
func (s *Server) release(w http.ResponseWriter, r *http.Request, u *auth.User) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	res, err := s.store.GetReservation(r.Context(), id)
	if err != nil {
		return err
	}
	if res.UserID != u.ID && (res.ProjectID != nil || res.DepartmentID != nil) {
		if err := s.checkEdit(u, projectOnly(res), res.DepartmentID); err != nil {
			return forbidden("only its department, the person who reserved it, or an admin can release it")
		}
	}
	if err := s.inv.Release(r.Context(), u.ID, id); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// projectOnly is the reservation's project when no department is set (a
// department reservation also stores its project).
func projectOnly(r db.Reservation) *int64 {
	if r.DepartmentID != nil {
		return nil
	}
	return r.ProjectID
}
