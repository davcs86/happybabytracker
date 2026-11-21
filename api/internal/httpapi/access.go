package httpapi

import (
	"errors"
	"net/http"

	"github.com/davcs86/happybabytracker/api/internal/auth"
	"github.com/davcs86/happybabytracker/api/internal/model"
	"github.com/davcs86/happybabytracker/api/internal/store"
)

type grantView struct {
	ParentID    string `json:"parent_id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
}

func (d Deps) listAccess(w http.ResponseWriter, r *http.Request) {
	grants, err := d.Store.AccessList(r.Context(), r.PathValue("kid"))
	if err != nil {
		respondInternal(w, err)
		return
	}
	rows := make([]grantView, len(grants))
	for i, g := range grants {
		rows[i] = grantView{ParentID: g.ParentID, Username: g.Username, DisplayName: g.DisplayName, Role: string(g.Role)}
	}
	respondJSON(w, http.StatusOK, struct {
		Grants []grantView `json:"grants"`
	}{rows})
}

// setAccess answers one 404 for a malformed, unknown or disabled username so none can be told apart.
func (d Deps) setAccess(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Role model.Role `json:"role"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	if in.Role != model.RoleGuardian && in.Role != model.RoleCaregiver {
		respondInvalid(w, "role")
		return
	}
	username, ok := auth.NormalizeUsername(r.PathValue("username"))
	if !ok {
		respondError(w, http.StatusNotFound, "parent_not_found")
		return
	}
	err := d.Store.SetAccess(r.Context(), r.PathValue("kid"), PrincipalFrom(r.Context()).ParentID, username, in.Role, d.Now())
	if errors.Is(err, store.ErrParentNotFound) {
		respondError(w, http.StatusNotFound, "parent_not_found")
		return
	}
	if kidWriteOK(w, err) {
		w.WriteHeader(http.StatusNoContent)
	}
}

// removeAccess lets anyone with a grant leave; only a guardian removes someone else.
func (d Deps) removeAccess(w http.ResponseWriter, r *http.Request) {
	target := r.PathValue("parent")
	if target != PrincipalFrom(r.Context()).ParentID && KidRoleFrom(r.Context()) != model.RoleGuardian {
		respondError(w, http.StatusForbidden, "forbidden")
		return
	}
	if kidWriteOK(w, d.Store.RemoveAccess(r.Context(), r.PathValue("kid"), PrincipalFrom(r.Context()).ParentID, target)) {
		w.WriteHeader(http.StatusNoContent)
	}
}

// kidWriteOK is true when err is nil; otherwise it answers the refused per-kid write and is false.
func kidWriteOK(w http.ResponseWriter, err error) bool {
	switch {
	case err == nil:
		return true
	case errors.Is(err, store.ErrNotFound):
		respondNoKid(w)
	case errors.Is(err, store.ErrForbidden):
		respondError(w, http.StatusForbidden, "forbidden")
	case errors.Is(err, store.ErrLastGuardian):
		respondError(w, http.StatusConflict, "last_guardian")
	default:
		respondInternal(w, err)
	}
	return false
}
