package dispatch

import (
	"net/http"
)

// ResourceHandlers specifies the handler for each RESTful action of a resource.
// Only non-nil handlers are registered as routes.
type ResourceHandlers struct {
	// Index handles GET /resources (collection listing).
	Index http.Handler
	// New handles GET /resources/new (new resource form).
	New http.Handler
	// Create handles POST /resources (create resource).
	Create http.Handler
	// Show handles GET /resources/{id} (display single resource).
	Show http.Handler
	// Edit handles GET /resources/{id}/edit (edit resource form).
	Edit http.Handler
	// Update handles PUT|PATCH /resources/{id} (update resource).
	Update http.Handler
	// Destroy handles DELETE /resources/{id} (delete resource).
	Destroy http.Handler
}

// ResourceOption is a functional option for configuring resource registration.
type ResourceOption func(*resourceConfig)

type resourceConfig struct {
	paramName    string
	excludePATCH bool
}

// WithParamName sets the URL parameter name used for member routes.
// Defaults to "id".
func WithParamName(name string) ResourceOption {
	return func(c *resourceConfig) { c.paramName = name }
}

// WithExcludePATCH prevents the PATCH method from being registered on the
// Update route. By default, Update matches both PUT and PATCH.
func WithExcludePATCH() ResourceOption {
	return func(c *resourceConfig) { c.excludePATCH = true }
}

// Resource registers standard RESTful routes for a plural resource.
// Only handlers that are non-nil in rh are registered.
//
// The registered routes follow Rails conventions:
//
//	GET    /<name>              -> <name>.index
//	GET    /<name>/new          -> <name>.new
//	POST   /<name>              -> <name>.create
//	GET    /<name>/{id}         -> <name>.show
//	GET    /<name>/{id}/edit    -> <name>.edit
//	PUT    /<name>/{id}         -> <name>.update   (also PATCH unless WithExcludePATCH)
//	DELETE /<name>/{id}         -> <name>.destroy
//
// Member routes (show, edit, update, destroy) include an Int constraint on
// the id parameter by default. Use WithParamName to change the parameter name.
func (r *Router) Resource(name string, rh ResourceHandlers, opts ...ResourceOption) error {
	cfg := resourceConfig{paramName: "id"}
	for _, o := range opts {
		o(&cfg)
	}

	basePath := "/" + name
	memberPath := basePath + "/{" + cfg.paramName + "}"
	memberConstraint := WithConstraint(Int(cfg.paramName))

	if rh.Index != nil {
		if err := r.GET(name+".index", basePath, rh.Index); err != nil {
			return err
		}
	}
	if rh.New != nil {
		if err := r.GET(name+".new", basePath+"/new", rh.New); err != nil {
			return err
		}
	}
	if rh.Create != nil {
		if err := r.POST(name+".create", basePath, rh.Create); err != nil {
			return err
		}
	}
	if rh.Show != nil {
		if err := r.GET(name+".show", memberPath, rh.Show, memberConstraint); err != nil {
			return err
		}
	}
	if rh.Edit != nil {
		if err := r.GET(name+".edit", memberPath+"/edit", rh.Edit, memberConstraint); err != nil {
			return err
		}
	}
	if rh.Update != nil {
		methods := PUT | PATCH
		if cfg.excludePATCH {
			methods = PUT
		}
		if err := r.registerMethod(methods, name+".update", memberPath, rh.Update, []RouteOption{memberConstraint}); err != nil {
			return err
		}
	}
	if rh.Destroy != nil {
		if err := r.DELETE(name+".destroy", memberPath, rh.Destroy, memberConstraint); err != nil {
			return err
		}
	}
	return nil
}

// SingularResource registers RESTful routes for a singular resource (no ID
// parameter). Only handlers that are non-nil in rh are registered.
//
// The registered routes follow Rails singular resource conventions:
//
//	GET    /<name>/new    -> <name>.new
//	POST   /<name>        -> <name>.create
//	GET    /<name>         -> <name>.show
//	GET    /<name>/edit    -> <name>.edit
//	PUT    /<name>         -> <name>.update   (also PATCH unless WithExcludePATCH)
//	DELETE /<name>         -> <name>.destroy
//
// Unlike plural resources, singular resources have no Index action and no
// {id} parameter in any route.
func (r *Router) SingularResource(name string, rh ResourceHandlers, opts ...ResourceOption) error {
	cfg := resourceConfig{paramName: "id"}
	for _, o := range opts {
		o(&cfg)
	}

	basePath := "/" + name

	if rh.Index != nil {
		// Singular resources do not support Index; ignore it silently
		// to match Rails behavior.
	}
	if rh.New != nil {
		if err := r.GET(name+".new", basePath+"/new", rh.New); err != nil {
			return err
		}
	}
	if rh.Create != nil {
		if err := r.POST(name+".create", basePath, rh.Create); err != nil {
			return err
		}
	}
	if rh.Show != nil {
		if err := r.GET(name+".show", basePath, rh.Show); err != nil {
			return err
		}
	}
	if rh.Edit != nil {
		if err := r.GET(name+".edit", basePath+"/edit", rh.Edit); err != nil {
			return err
		}
	}
	if rh.Update != nil {
		methods := PUT | PATCH
		if cfg.excludePATCH {
			methods = PUT
		}
		if err := r.registerMethod(methods, name+".update", basePath, rh.Update, nil); err != nil {
			return err
		}
	}
	if rh.Destroy != nil {
		if err := r.DELETE(name+".destroy", basePath, rh.Destroy); err != nil {
			return err
		}
	}
	return nil
}
