package httpapi

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/auth"
	"github.com/davcs86/happybabytracker/api/internal/config"
	"github.com/davcs86/happybabytracker/api/internal/store"
)

// Access is the one class a route declares; the middleware enforces it.
type Access int

const (
	Open      Access = iota // no token
	Internal                // internal token only
	Session                 // any valid session
	Admin                   // admin session
	Parent                  // parent session
	Caregiver               // parent with any role on {kid}
	Guardian                // parent with the guardian role on {kid}
)

type Route struct {
	Method, Pattern string
	Access          Access
	Handler         http.HandlerFunc
}

type Deps struct {
	Store   *store.Store
	Cfg     config.Config
	Limiter *auth.Limiter
	Now     func() time.Time
	Log     *slog.Logger
}

// Routes is the only place routes are declared.
func Routes(d Deps) []Route {
	return []Route{
		{http.MethodGet, "/healthz", Open, Health(d.Store)},
		{http.MethodPost, "/v1/session", Internal, d.login},
		{http.MethodGet, "/v1/session", Session, d.currentSession},
		{http.MethodDelete, "/v1/session", Session, d.logout},

		{http.MethodPost, "/v1/admin/parents", Admin, d.createParent},
		{http.MethodGet, "/v1/admin/parents", Admin, d.listParents},
		{http.MethodPost, "/v1/admin/parents/{parent}/reset-link", Admin, d.resetLink},
		{http.MethodPost, "/v1/admin/parents/{parent}/disable", Admin, d.setDisabled(true)},
		{http.MethodPost, "/v1/admin/parents/{parent}/enable", Admin, d.setDisabled(false)},

		{http.MethodGet, "/v1/tokens/{token}", Internal, d.peekToken},
		{http.MethodPost, "/v1/tokens/{token}/redeem", Internal, d.redeemToken},

		{http.MethodPatch, "/v1/me", Parent, d.updateProfile},
		{http.MethodPost, "/v1/me/password", Parent, d.changePassword},
		{http.MethodGet, "/v1/me/sessions", Parent, d.listSessions},
		{http.MethodDelete, "/v1/me/sessions/{session}", Parent, d.deleteSession},

		{http.MethodGet, "/v1/growth-sources", Parent, d.growthSources},
		{http.MethodGet, "/v1/vaccine-schemas", Parent, d.vaccineSchemas},
		{http.MethodGet, "/v1/vaccine-catalogue", Parent, d.vaccineCatalogue},

		{http.MethodGet, "/v1/kids", Parent, d.listKids},
		{http.MethodPost, "/v1/kids", Parent, d.createKid},
		{http.MethodGet, "/v1/kids/{kid}", Caregiver, d.getKid},
		{http.MethodPatch, "/v1/kids/{kid}", Guardian, d.updateKid},
		{http.MethodDelete, "/v1/kids/{kid}", Guardian, d.deleteKid},
		{http.MethodGet, "/v1/kids/{kid}/access", Guardian, d.listAccess},
		{http.MethodPut, "/v1/kids/{kid}/access/{username}", Guardian, d.setAccess},
		{http.MethodDelete, "/v1/kids/{kid}/access/{parent}", Caregiver, d.removeAccess},

		{http.MethodGet, "/v1/kids/{kid}/bottles", Caregiver, d.listBottles},
		{http.MethodPost, "/v1/kids/{kid}/bottles", Caregiver, d.createBottle},
		{http.MethodGet, "/v1/kids/{kid}/bottles/{entry}", Caregiver, d.getBottle},
		{http.MethodPatch, "/v1/kids/{kid}/bottles/{entry}", Caregiver, d.updateBottle},
		{http.MethodDelete, "/v1/kids/{kid}/bottles/{entry}", Caregiver, d.deleteBottle},

		{http.MethodGet, "/v1/kids/{kid}/diapers", Caregiver, d.listDiapers},
		{http.MethodPost, "/v1/kids/{kid}/diapers", Caregiver, d.createDiaper},
		{http.MethodGet, "/v1/kids/{kid}/diapers/{entry}", Caregiver, d.getDiaper},
		{http.MethodPatch, "/v1/kids/{kid}/diapers/{entry}", Caregiver, d.updateDiaper},
		{http.MethodDelete, "/v1/kids/{kid}/diapers/{entry}", Caregiver, d.deleteDiaper},

		{http.MethodGet, "/v1/kids/{kid}/meals", Caregiver, d.listMeals},
		{http.MethodPost, "/v1/kids/{kid}/meals", Caregiver, d.createMeal},
		{http.MethodGet, "/v1/kids/{kid}/meals/{entry}", Caregiver, d.getMeal},
		{http.MethodPatch, "/v1/kids/{kid}/meals/{entry}", Caregiver, d.updateMeal},
		{http.MethodDelete, "/v1/kids/{kid}/meals/{entry}", Caregiver, d.deleteMeal},

		{http.MethodGet, "/v1/kids/{kid}/pumping", Caregiver, d.listPumping},
		{http.MethodPost, "/v1/kids/{kid}/pumping", Caregiver, d.createPumping},
		{http.MethodGet, "/v1/kids/{kid}/pumping/{entry}", Caregiver, d.getPumping},
		{http.MethodPatch, "/v1/kids/{kid}/pumping/{entry}", Caregiver, d.updatePumping},
		{http.MethodDelete, "/v1/kids/{kid}/pumping/{entry}", Caregiver, d.deletePumping},

		{http.MethodGet, "/v1/kids/{kid}/sleep", Caregiver, d.listSleep},
		{http.MethodPost, "/v1/kids/{kid}/sleep", Caregiver, d.createSleep},
		{http.MethodGet, "/v1/kids/{kid}/sleep/{entry}", Caregiver, d.getSleep},
		{http.MethodPatch, "/v1/kids/{kid}/sleep/{entry}", Caregiver, d.updateSleep},
		{http.MethodDelete, "/v1/kids/{kid}/sleep/{entry}", Caregiver, d.deleteSleep},
		{http.MethodPost, "/v1/kids/{kid}/sleep/{entry}/end", Caregiver, d.endSleep},

		{http.MethodGet, "/v1/kids/{kid}/nursing", Caregiver, d.listNursing},
		{http.MethodPost, "/v1/kids/{kid}/nursing", Caregiver, d.createNursing},
		{http.MethodGet, "/v1/kids/{kid}/nursing/{entry}", Caregiver, d.getNursing},
		{http.MethodPatch, "/v1/kids/{kid}/nursing/{entry}", Caregiver, d.updateNursing},
		{http.MethodDelete, "/v1/kids/{kid}/nursing/{entry}", Caregiver, d.deleteNursing},
		{http.MethodPost, "/v1/kids/{kid}/nursing/{entry}/end", Caregiver, d.endNursing},
		{http.MethodPost, "/v1/kids/{kid}/nursing/{entry}/side", Caregiver, d.switchNursingSide},

		{http.MethodGet, "/v1/kids/{kid}/growth", Caregiver, d.listGrowth},
		{http.MethodPost, "/v1/kids/{kid}/growth", Caregiver, d.createGrowth},
		{http.MethodGet, "/v1/kids/{kid}/growth/curves", Caregiver, d.growthCurves},
		{http.MethodGet, "/v1/kids/{kid}/growth/{entry}", Caregiver, d.getGrowth},
		{http.MethodPatch, "/v1/kids/{kid}/growth/{entry}", Caregiver, d.updateGrowth},
		{http.MethodDelete, "/v1/kids/{kid}/growth/{entry}", Caregiver, d.deleteGrowth},

		{http.MethodGet, "/v1/kids/{kid}/symptoms", Caregiver, d.listSymptoms},
		{http.MethodPost, "/v1/kids/{kid}/symptoms", Caregiver, d.createSymptom},
		{http.MethodGet, "/v1/kids/{kid}/symptoms/{entry}", Caregiver, d.getSymptom},
		{http.MethodPatch, "/v1/kids/{kid}/symptoms/{entry}", Caregiver, d.updateSymptom},
		{http.MethodDelete, "/v1/kids/{kid}/symptoms/{entry}", Caregiver, d.deleteSymptom},

		{http.MethodGet, "/v1/kids/{kid}/medications", Caregiver, d.listMedications},
		{http.MethodPost, "/v1/kids/{kid}/medications", Caregiver, d.createMedication},
		{http.MethodGet, "/v1/kids/{kid}/medications/{entry}", Caregiver, d.getMedication},
		{http.MethodPatch, "/v1/kids/{kid}/medications/{entry}", Caregiver, d.updateMedication},
		{http.MethodDelete, "/v1/kids/{kid}/medications/{entry}", Caregiver, d.deleteMedication},

		{http.MethodGet, "/v1/kids/{kid}/vaccines", Caregiver, d.kidVaccines},
		{http.MethodGet, "/v1/kids/{kid}/vaccinations", Caregiver, d.listVaccinations},
		{http.MethodPost, "/v1/kids/{kid}/vaccinations", Caregiver, d.createVaccination},
		{http.MethodGet, "/v1/kids/{kid}/vaccinations/{administration}", Caregiver, d.getVaccination},
		{http.MethodPatch, "/v1/kids/{kid}/vaccinations/{administration}", Caregiver, d.updateVaccination},
		{http.MethodDelete, "/v1/kids/{kid}/vaccinations/{administration}", Caregiver, d.deleteVaccination},
		{http.MethodPut, "/v1/kids/{kid}/vaccine-dismissals/{line}", Guardian, d.setDismissal},
		{http.MethodDelete, "/v1/kids/{kid}/vaccine-dismissals/{line}", Guardian, d.clearDismissal},
		{http.MethodPost, "/v1/kids/{kid}/vaccine-dismissals", Guardian, d.dismissBefore},

		{http.MethodGet, "/v1/kids/{kid}/timeline", Caregiver, d.timeline},
		{http.MethodGet, "/v1/kids/{kid}/summary", Caregiver, d.summary},
		{http.MethodGet, "/v1/kids/{kid}/modules", Caregiver, d.listModules},
		{http.MethodPut, "/v1/kids/{kid}/modules/{module}", Guardian, d.setModule},
		{http.MethodGet, "/v1/kids/{kid}/home", Caregiver, d.home},
	}
}

func NewHandler(d Deps) http.Handler {
	return newMux(d, Routes(d))
}

// newMux sends every request, matched or not, through the middleware.
func newMux(d Deps, routes []Route) http.Handler {
	mux := http.NewServeMux()
	for _, rt := range routes {
		mux.Handle(rt.Method+" "+rt.Pattern, d.guard(rt.Method+" "+rt.Pattern, rt.Access, rt.Handler))
	}
	mux.Handle("/", d.guard("unmatched", Internal, func(w http.ResponseWriter, _ *http.Request) {
		respondError(w, http.StatusNotFound, "not_found")
	}))
	return mux
}
