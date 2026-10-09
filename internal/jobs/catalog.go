package jobs

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
		r("chef", "Chef"), r("sous-chef", "Sous Chef"), r("line-cook", "Line Cook"),
		r("prep-assistant", "Prep Assistant"), r("kitchen-helper", "Kitchen Helper"),
		r("dishwasher", "Dishwasher"), r("pastry-chef", "Pastry Chef"), r("grill-cook", "Grill Cook"),
		r("barista", "Barista"), r("bartender", "Bartender"), r("food-runner", "Food Runner"),
		r("buffet-server", "Buffet Server"), r("kitchen-steward", "Kitchen Steward"),
		r("catering-assistant", "Catering Assistant"),
	}},
	{"delivery", "Delivery", []Role{
		r("motorcycle", "Motorcycle Rider"), r("food-delivery", "Food Delivery Rider"),
		r("car-driver", "Car Driver"), r("bicycle", "Bicycle Courier"),
		r("escooter", "E-Scooter Courier"), r("van-driver", "Van Driver"),
		r("express-messenger", "Express Messenger"), r("pickup-driver", "Pickup Truck Driver"),
		r("truck-driver", "Heavy Truck Driver"), r("document-courier", "Document Courier"),
		r("dispatch-driver", "Warehouse Dispatch Driver"), r("last-mile", "Last-Mile Delivery Specialist"),
	}},
	{"helpers", "Helpers", []Role{
		r("event", "Event Helper"), r("banquet", "Banquet Server"), r("warehouse", "Warehouse Helper"),
		r("inventory", "Inventory & Stock Assistant"), r("loading-crew", "Loading & Unloading Crew"),
		r("office", "Office Helper"), r("moving", "Moving Helper"), r("cleaning", "Cleaning Crew"),
		r("sanitation", "Sanitation Staff"), r("staging", "Staging & Booth Builder"),
		r("general-laborer", "General Laborer"), r("retail-assistant", "Retail Store Assistant"),
		r("maintenance", "Maintenance Assistant"), r("usher-assistant", "Security / Usher Assistant"),
	}},
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
