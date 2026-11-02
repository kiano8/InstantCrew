package jobs

import (
	"context"

	"instantcrew/internal/store"
)

type Role struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

type Category struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Roles []Role `json:"roles"`
}

func r(id, label string) Role { return Role{id, label} }

var Catalog = []Category{
	{"kitchen", "Kitchen", []Role{
		r("all-around-kitchen-crew", "All-around Kitchen Crew"),
		r("cook", "Cook"),
		r("kitchen-helper", "Kitchen Helper"),
		r("baker", "Baker"),
		r("dishwasher", "Dishwasher"),
		r("all-around-service-crew", "All-around Service Crew"),
		r("waiter-server", "Waiter / Server"),
		r("barista", "Barista"),
		r("cashier", "Cashier"),
		r("store-manager", "Store Manager"),
		r("janitorial-cleaning-staff", "Janitorial / Cleaning Staff"),
	}},
	{"delivery", "Delivery", []Role{
		r("2-wheel-driver", "2-Wheel Driver"),
		r("suv-van-driver", "SUV / Van Driver"),
		r("4-10-wheel-truck-driver", "4-10 Wheel Truck Driver"),
		r("warehouse-picker-packer", "Warehouse Picker / Packer"),
		r("kargador-loader-unloader", "Kargador / Loader / Unloader"),
		r("forklift-operator", "Forklift Operator"),
	}},
	{"helpers", "Helper", []Role{
		r("construction-worker", "Construction Worker / Laborer"),
		r("electrician", "Electrician"),
		r("plumber", "Plumber"),
		r("welder", "Welder"),
		r("carpenter-painter", "Carpenter / Painter"),
	}},
	{"personal", "Personal", []Role{
		r("caregiver", "Caregiver"),
		r("house-helper", "House Helper"),
		r("nutritionist", "Nutritionist"),
		r("fitness-trainer", "Fitness Trainer"),
		r("yoga-instructor", "Yoga Instructor"),
		r("makeup-artist", "Make-up Artist"),
		r("hair-stylist", "Hair Stylist"),
		r("massage-therapist", "Massage Therapist"),
		r("physical-therapist", "Physical Therapist"),
	}},
	{"education", "Education", []Role{
		r("math-tutor-g1-g6", "Math Tutor (G1-G6)"),
		r("math-tutor-h1-h4", "Math Tutor (H1-H4)"),
		r("chinese-tutor-g1-g6", "Chinese Tutor (G1-G6)"),
		r("chinese-tutor-h1-h4", "Chinese Tutor (H1-H4)"),
		r("art-teacher-kids", "Art Teacher (5-14 yrs old)"),
		r("music-teacher-kids", "Music Teacher (5-14 yrs old)"),
	}},
	{"creative", "Creative", []Role{
		r("graphic-designer", "Graphic Designer"),
		r("video-editor", "Video Editor"),
		r("photographer", "Photographer"),
		r("videographer", "Videographer"),
		r("social-media-manager", "Social Media Manager"),
		r("dj", "DJ"),
		r("host", "Host"),
	}},
}

type Doc struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Group string `json:"group"`
}

// RequirementDocs are the documents an employer can require; ids are stored on
// jobs.required_docs, so never rename one. The employer picker groups by Group.
var RequirementDocs = []Doc{
	{"gov-id", "Government-issued ID", "Valid IDs"},
	{"drivers-license", "Driver's License", "Valid IDs"},
	{"passport", "Passport", "Valid IDs"},
	{"nbi", "NBI Clearance", "Clearances"},
	{"barangay-clearance", "Barangay Clearance", "Clearances"},
	{"police-clearance", "Police Clearance", "Clearances"},
	{"good-moral", "Certificate of Good Moral Character", "Clearances"},
	{"health-card", "Health Certificate / Health Card", "Health"},
	{"medical-cert", "Medical Certificate", "Health"},
	{"resume", "Resume", "Work history & credentials"},
	{"employment-cert", "Certificate of Previous Employment", "Work history & credentials"},
	{"training-cert", "Training Certificate", "Work history & credentials"},
	{"prc-license", "PRC Professional License", "Work history & credentials"},
	{"pagibig", "Pag-IBIG", "Government benefits"},
	{"philhealth", "PhilHealth", "Government benefits"},
}

func RoleByID(id string) (Role, bool) {
	for _, c := range Catalog {
		for _, role := range c.Roles {
			if role.ID == id {
				return role, true
			}
		}
	}
	return Role{}, false
}

// SyncCatalog copies the Go catalog into job_roles at startup. The Go list is the source of truth.
func SyncCatalog(ctx context.Context, st *store.Store) error {
	var rows []store.RoleRow
	for _, c := range Catalog {
		for _, role := range c.Roles {
			rows = append(rows, store.RoleRow{ID: role.ID, Category: c.ID, Label: role.Label})
		}
	}
	return st.SyncRoles(ctx, rows)
}

func LookupRole(category, roleID string) (Role, bool) {
	for _, c := range Catalog {
		if c.ID != category {
			continue
		}
		for _, role := range c.Roles {
			if role.ID == roleID {
				return role, true
			}
		}
	}
	return Role{}, false
}
