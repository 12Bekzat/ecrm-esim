package httpapi

import (
	"context"
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"ecrm/backend/internal/auth"
	"ecrm/backend/internal/providers/zesimo"
)

func pageParams(r *http.Request) (int, int) {
	page := auth.ParseLimit(r.URL.Query().Get("page"), 1, 1000000)
	limit := auth.ParseLimit(r.URL.Query().Get("limit"), 25, 100)
	return page, limit
}
func listResponse(w http.ResponseWriter, data any, meta Pagination) {
	writeJSONStatus(w, http.StatusOK, map[string]any{"data": data, "meta": meta})
}
func dateRange(r *http.Request) (time.Time, time.Time, error) {
	now := time.Now().UTC()
	from := now.AddDate(0, 0, -30)
	to := now.AddDate(0, 0, 1)
	var e error
	if v := r.URL.Query().Get("from"); v != "" {
		from, e = time.Parse("2006-01-02", v)
		if e != nil {
			return from, to, e
		}
	}
	if v := r.URL.Query().Get("to"); v != "" {
		to, e = time.Parse("2006-01-02", v)
		if e != nil {
			return from, to, e
		}
		to = to.AddDate(0, 0, 1)
	}
	if !from.Before(to) {
		return from, to, fmt.Errorf("from must be before to")
	}
	if to.Sub(from) > 366*24*time.Hour {
		return from, to, fmt.Errorf("date range cannot exceed 366 days")
	}
	return from, to, nil
}
func (s *Server) audit(r *http.Request, action, entity, id string, old, new any) {
	s.store.Audit(r.Context(), currentUser(r), action, entity, id, old, new, clientIP(r))
}

func (s *Server) dashboard(w http.ResponseWriter, r *http.Request) {
	from, to, e := dateRange(r)
	if e != nil {
		writeError(w, http.StatusUnprocessableEntity, "validation_error", e.Error(), nil)
		return
	}
	x, e := s.store.Dashboard(r.Context(), from, to)
	if e != nil {
		respondStoreError(w, e)
		return
	}
	writeJSON(w, http.StatusOK, x)
}
func (s *Server) search(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if len(q) < 2 {
		writeJSON(w, http.StatusOK, []SearchItem{})
		return
	}
	items, e := s.store.Search(r.Context(), q, auth.ParseLimit(r.URL.Query().Get("limit"), 10, 30))
	if e != nil {
		respondStoreError(w, e)
		return
	}
	writeJSON(w, http.StatusOK, items)
}
func (s *Server) customerStats(w http.ResponseWriter, r *http.Request) {
	x, e := s.store.CustomerStats(r.Context())
	if e != nil {
		respondStoreError(w, e)
		return
	}
	writeJSON(w, http.StatusOK, x)
}
func (s *Server) esimStats(w http.ResponseWriter, r *http.Request) {
	x, e := s.store.ESIMStats(r.Context())
	if e != nil {
		respondStoreError(w, e)
		return
	}
	writeJSON(w, http.StatusOK, x)
}

func (s *Server) listCustomers(w http.ResponseWriter, r *http.Request) {
	p, l := pageParams(r)
	x, m, e := s.store.ListCustomers(r.Context(), r.URL.Query().Get("q"), r.URL.Query().Get("status"), p, l)
	if e != nil {
		respondStoreError(w, e)
		return
	}
	listResponse(w, x, m)
}
func (s *Server) getCustomer(w http.ResponseWriter, r *http.Request) {
	x, e := s.store.CustomerByID(r.Context(), r.PathValue("id"))
	if e != nil {
		respondStoreError(w, e)
		return
	}
	writeJSON(w, http.StatusOK, x)
}
func (s *Server) createCustomer(w http.ResponseWriter, r *http.Request) {
	var in CustomerInput
	if !decodeJSON(w, r, &in) {
		return
	}
	if in.Name == "" || in.Phone == "" {
		writeError(w, 422, "validation_error", "name and phone are required", nil)
		return
	}
	x, e := s.store.CreateCustomer(r.Context(), in)
	if e != nil {
		respondStoreError(w, e)
		return
	}
	s.audit(r, "create", "customer", x.ID, nil, x)
	writeJSON(w, http.StatusCreated, x)
}
func (s *Server) updateCustomer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	old, e := s.store.CustomerByID(r.Context(), id)
	if e != nil {
		respondStoreError(w, e)
		return
	}
	var in CustomerInput
	if !decodeJSON(w, r, &in) {
		return
	}
	x, e := s.store.UpdateCustomer(r.Context(), id, in)
	if e != nil {
		respondStoreError(w, e)
		return
	}
	s.audit(r, "update", "customer", id, old, x)
	writeJSON(w, 200, x)
}
func (s *Server) deleteCustomer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	old, e := s.store.CustomerByID(r.Context(), id)
	if e != nil {
		respondStoreError(w, e)
		return
	}
	if e = s.store.DeleteCustomer(r.Context(), id); e != nil {
		respondStoreError(w, e)
		return
	}
	s.audit(r, "delete", "customer", id, old, nil)
	w.WriteHeader(204)
}

func (s *Server) listSuppliers(w http.ResponseWriter, r *http.Request) {
	p, l := pageParams(r)
	x, m, e := s.store.ListSuppliers(r.Context(), r.URL.Query().Get("q"), p, l)
	if e != nil {
		respondStoreError(w, e)
		return
	}
	listResponse(w, x, m)
}
func (s *Server) getSupplier(w http.ResponseWriter, r *http.Request) {
	x, e := s.store.SupplierByID(r.Context(), r.PathValue("id"))
	if e != nil {
		respondStoreError(w, e)
		return
	}
	writeJSON(w, 200, x)
}
func (s *Server) createSupplier(w http.ResponseWriter, r *http.Request) {
	var in SupplierInput
	if !decodeJSON(w, r, &in) {
		return
	}
	if in.Name == "" {
		writeError(w, 422, "validation_error", "name is required", nil)
		return
	}
	x, e := s.store.CreateSupplier(r.Context(), in)
	if e != nil {
		respondStoreError(w, e)
		return
	}
	s.audit(r, "create", "supplier", x.ID, nil, x)
	writeJSON(w, 201, x)
}
func (s *Server) updateSupplier(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	old, e := s.store.SupplierByID(r.Context(), id)
	if e != nil {
		respondStoreError(w, e)
		return
	}
	var in SupplierInput
	if !decodeJSON(w, r, &in) {
		return
	}
	x, e := s.store.UpdateSupplier(r.Context(), id, in)
	if e != nil {
		respondStoreError(w, e)
		return
	}
	s.audit(r, "update", "supplier", id, old, x)
	writeJSON(w, 200, x)
}
func (s *Server) deleteSupplier(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	old, e := s.store.SupplierByID(r.Context(), id)
	if e != nil {
		respondStoreError(w, e)
		return
	}
	if e = s.store.DeleteSupplier(r.Context(), id); e != nil {
		respondStoreError(w, e)
		return
	}
	s.audit(r, "delete", "supplier", id, old, nil)
	w.WriteHeader(204)
}
func (s *Server) syncSupplier(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	supplier, lookupErr := s.store.SupplierByID(r.Context(), id)
	if lookupErr != nil {
		respondStoreError(w, lookupErr)
		return
	}
	if supplier.Name == zesimoSupplierName {
		result, e := s.store.SyncZesimoCatalog(r.Context(), s.zesimo)
		if e != nil {
			writeError(w, http.StatusBadGateway, "zesimo_error", e.Error(), nil)
			return
		}
		s.audit(r, "sync", "supplier", id, nil, result)
		writeJSON(w, 200, result)
		return
	}
	if e := s.store.SyncSupplier(r.Context(), id); e != nil {
		respondStoreError(w, e)
		return
	}
	x, e := s.store.SupplierByID(r.Context(), id)
	if e != nil {
		respondStoreError(w, e)
		return
	}
	s.audit(r, "sync", "supplier", id, nil, x)
	writeJSON(w, 200, x)
}
func (s *Server) syncAllSuppliers(w http.ResponseWriter, r *http.Request) {
	items, _, e := s.store.ListSuppliers(r.Context(), "", 1, 100)
	if e != nil {
		respondStoreError(w, e)
		return
	}
	synced := 0
	failures := map[string]string{}
	for _, x := range items {
		if x.Name == zesimoSupplierName {
			_, e = s.store.SyncZesimoCatalog(r.Context(), s.zesimo)
		} else {
			e = s.store.SyncSupplier(r.Context(), x.ID)
		}
		if e != nil {
			failures[x.ID] = e.Error()
		} else {
			synced++
		}
	}
	s.audit(r, "sync_all", "supplier", "", nil, map[string]int{"synced": synced})
	writeJSON(w, 200, map[string]any{"synced": synced, "failures": failures})
}

func (s *Server) listTariffs(w http.ResponseWriter, r *http.Request) {
	p, l := pageParams(r)
	q := r.URL.Query()
	x, m, e := s.store.ListTariffs(r.Context(), q.Get("q"), q.Get("country"), q.Get("supplierId"), q.Get("status"), p, l)
	if e != nil {
		respondStoreError(w, e)
		return
	}
	listResponse(w, x, m)
}
func (s *Server) getTariff(w http.ResponseWriter, r *http.Request) {
	x, e := s.store.TariffByID(r.Context(), r.PathValue("id"))
	if e != nil {
		respondStoreError(w, e)
		return
	}
	writeJSON(w, 200, x)
}
func validateTariff(in TariffInput) error {
	if in.SupplierID == "" || in.Name == "" || in.Country == "" {
		return fmt.Errorf("supplierId, name and country are required")
	}
	if in.DataMB <= 0 || in.ValidityDays <= 0 {
		return fmt.Errorf("dataMb and validityDays must be positive")
	}
	if in.Price < 0 || in.Cost < 0 {
		return fmt.Errorf("prices cannot be negative")
	}
	return nil
}
func (s *Server) createTariff(w http.ResponseWriter, r *http.Request) {
	var in TariffInput
	if !decodeJSON(w, r, &in) {
		return
	}
	if e := validateTariff(in); e != nil {
		writeError(w, 422, "validation_error", e.Error(), nil)
		return
	}
	x, e := s.store.CreateTariff(r.Context(), in)
	if e != nil {
		respondStoreError(w, e)
		return
	}
	s.audit(r, "create", "tariff", x.ID, nil, x)
	writeJSON(w, 201, x)
}
func (s *Server) updateTariff(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	old, e := s.store.TariffByID(r.Context(), id)
	if e != nil {
		respondStoreError(w, e)
		return
	}
	var in TariffInput
	if !decodeJSON(w, r, &in) {
		return
	}
	if e = validateTariff(in); e != nil {
		writeError(w, 422, "validation_error", e.Error(), nil)
		return
	}
	x, e := s.store.UpdateTariff(r.Context(), id, in)
	if e != nil {
		respondStoreError(w, e)
		return
	}
	s.audit(r, "update", "tariff", id, old, x)
	writeJSON(w, 200, x)
}
func (s *Server) deleteTariff(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	old, e := s.store.TariffByID(r.Context(), id)
	if e != nil {
		respondStoreError(w, e)
		return
	}
	if e = s.store.DeleteTariff(r.Context(), id); e != nil {
		respondStoreError(w, e)
		return
	}
	s.audit(r, "delete", "tariff", id, old, nil)
	w.WriteHeader(204)
}

func (s *Server) listESIMs(w http.ResponseWriter, r *http.Request) {
	p, l := pageParams(r)
	q := r.URL.Query()
	x, m, e := s.store.ListESIMs(r.Context(), q.Get("q"), q.Get("status"), q.Get("supplierId"), q.Get("tariffId"), p, l)
	if e != nil {
		respondStoreError(w, e)
		return
	}
	for i := range x {
		if currentUser(r).Role == "viewer" {
			x[i].ActivationCode = ""
			x[i].QRPayload = ""
		}
	}
	listResponse(w, x, m)
}
func (s *Server) getESIM(w http.ResponseWriter, r *http.Request) {
	x, e := s.store.ESIMByID(r.Context(), r.PathValue("id"))
	if e != nil {
		respondStoreError(w, e)
		return
	}
	writeJSON(w, 200, x)
}
func validateESIM(in ESIMInput) error {
	if in.SupplierID == "" || strings.TrimSpace(in.ICCID) == "" {
		return fmt.Errorf("supplierId and iccid are required")
	}
	return nil
}
func (s *Server) createESIM(w http.ResponseWriter, r *http.Request) {
	var in ESIMInput
	if !decodeJSON(w, r, &in) {
		return
	}
	if e := validateESIM(in); e != nil {
		writeError(w, 422, "validation_error", e.Error(), nil)
		return
	}
	x, e := s.store.CreateESIM(r.Context(), in)
	if e != nil {
		respondStoreError(w, e)
		return
	}
	s.audit(r, "create", "esim", x.ID, nil, x)
	writeJSON(w, 201, x)
}
func (s *Server) updateESIM(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	old, e := s.store.ESIMByID(r.Context(), id)
	if e != nil {
		respondStoreError(w, e)
		return
	}
	var in ESIMInput
	if !decodeJSON(w, r, &in) {
		return
	}
	x, e := s.store.UpdateESIM(r.Context(), id, in)
	if e != nil {
		respondStoreError(w, e)
		return
	}
	s.audit(r, "update", "esim", id, old, x)
	writeJSON(w, 200, x)
}
func (s *Server) deleteESIM(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	old, e := s.store.ESIMByID(r.Context(), id)
	if e != nil {
		respondStoreError(w, e)
		return
	}
	if e = s.store.DeleteESIM(r.Context(), id); e != nil {
		respondStoreError(w, e)
		return
	}
	s.audit(r, "delete", "esim", id, old, nil)
	w.WriteHeader(204)
}
func (s *Server) importESIMs(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Items []ESIMInput `json:"items"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	if len(in.Items) == 0 || len(in.Items) > 5000 {
		writeError(w, 422, "validation_error", "items must contain 1 to 5000 records", nil)
		return
	}
	n, dups, e := s.store.ImportESIMs(r.Context(), in.Items)
	if e != nil {
		respondStoreError(w, e)
		return
	}
	s.audit(r, "import", "esim", "", nil, map[string]any{"created": n, "duplicates": len(dups)})
	writeJSON(w, 201, map[string]any{"created": n, "duplicates": dups})
}

func (s *Server) listOrders(w http.ResponseWriter, r *http.Request) {
	p, l := pageParams(r)
	q := r.URL.Query()
	var from, to *time.Time
	if v := q.Get("from"); v != "" {
		x, e := time.Parse("2006-01-02", v)
		if e != nil {
			writeError(w, 422, "validation_error", "invalid from date", nil)
			return
		}
		from = &x
	}
	if v := q.Get("to"); v != "" {
		x, e := time.Parse("2006-01-02", v)
		if e != nil {
			writeError(w, 422, "validation_error", "invalid to date", nil)
			return
		}
		x = x.AddDate(0, 0, 1)
		to = &x
	}
	x, m, e := s.store.ListOrders(r.Context(), q.Get("q"), q.Get("status"), q.Get("paymentStatus"), q.Get("customerId"), from, to, p, l)
	if e != nil {
		respondStoreError(w, e)
		return
	}
	listResponse(w, x, m)
}
func (s *Server) getOrder(w http.ResponseWriter, r *http.Request) {
	x, e := s.store.OrderByID(r.Context(), r.PathValue("id"))
	if e != nil {
		respondStoreError(w, e)
		return
	}
	writeJSON(w, 200, x)
}
func (s *Server) createOrder(w http.ResponseWriter, r *http.Request) {
	var in OrderCreateInput
	if !decodeJSON(w, r, &in) {
		return
	}
	if in.TariffID == "" {
		writeError(w, 422, "validation_error", "tariffId is required", nil)
		return
	}
	x, e := s.store.CreateOrder(r.Context(), in, currentUser(r).ID)
	if e != nil {
		respondStoreError(w, e)
		return
	}
	s.audit(r, "create", "order", x.ID, nil, x)
	if x.ProvisioningStatus == "pending" && x.PaymentStatus == "paid" {
		var complete bool
		x, complete, e = s.provisionZesimoOrder(r.Context(), x)
		if e != nil {
			writeError(w, http.StatusBadGateway, "zesimo_provisioning_failed", e.Error(), map[string]any{"orderId": x.ID, "retry": "POST /api/v1/orders/" + x.ID + "/provision"})
			return
		}
		if !complete {
			writeJSON(w, http.StatusAccepted, x)
			return
		}
	}
	writeJSON(w, 201, x)
}
func (s *Server) updateOrder(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	old, e := s.store.OrderByID(r.Context(), id)
	if e != nil {
		respondStoreError(w, e)
		return
	}
	var in OrderUpdateInput
	if !decodeJSON(w, r, &in) {
		return
	}
	x, e := s.store.UpdateOrder(r.Context(), id, in)
	if e != nil {
		respondStoreError(w, e)
		return
	}
	if x.ESIMID == nil && x.PaymentStatus == "paid" && x.ProvisioningStatus == "pending" {
		x, _, e = s.provisionZesimoOrder(r.Context(), x)
		if e != nil {
			writeError(w, http.StatusBadGateway, "zesimo_provisioning_failed", e.Error(), map[string]any{"orderId": x.ID})
			return
		}
	}
	s.audit(r, "update", "order", id, old, x)
	writeJSON(w, 200, x)
}

func (s *Server) provisionOrder(w http.ResponseWriter, r *http.Request) {
	order, e := s.store.OrderByID(r.Context(), r.PathValue("id"))
	if e != nil {
		respondStoreError(w, e)
		return
	}
	if order.PaymentStatus != "paid" {
		writeError(w, 422, "payment_required", "Order must be paid before provisioning", nil)
		return
	}
	if order.ESIMID != nil {
		writeJSON(w, 200, order)
		return
	}
	order, complete, e := s.provisionZesimoOrder(r.Context(), order)
	if e != nil {
		writeError(w, http.StatusBadGateway, "zesimo_provisioning_failed", e.Error(), map[string]any{"orderId": order.ID})
		return
	}
	s.audit(r, "provision", "order", order.ID, nil, order)
	if !complete {
		writeJSON(w, http.StatusAccepted, order)
		return
	}
	writeJSON(w, 200, order)
}

func (s *Server) provisionZesimoOrder(ctx context.Context, order Order) (Order, bool, error) {
	if s.zesimo == nil || !s.zesimo.Configured() {
		return order, false, fmt.Errorf("Zesimo API key is not configured")
	}
	packageCode, key, e := s.store.PrepareZesimoOrder(ctx, order.ID)
	if e != nil {
		return order, false, e
	}
	var externalOrder zesimo.Order
	if order.ProviderOrderID != nil && *order.ProviderOrderID != "" {
		externalOrder, e = s.zesimo.Order(ctx, *order.ProviderOrderID)
	} else {
		externalOrder, e = s.zesimo.CreateOrder(ctx, packageCode, 1, key)
	}
	if e != nil {
		s.store.FailZesimoProvisioning(ctx, order.ID, e)
		return order, false, e
	}
	if e = s.store.RecordZesimoProvisioning(ctx, order.ID, externalOrder); e != nil {
		return order, false, e
	}
	if len(externalOrder.ESIMs) == 0 {
		updated, _ := s.store.OrderByID(ctx, order.ID)
		return updated, false, nil
	}
	updated, e := s.store.CompleteZesimoProvisioning(ctx, order.ID, externalOrder)
	return updated, e == nil, e
}

func (s *Server) zesimoStatus(w http.ResponseWriter, r *http.Request) {
	response := map[string]any{"configured": s.zesimo != nil && s.zesimo.Configured(), "keyName": s.cfg.ZesimoKeyName, "baseUrl": s.cfg.ZesimoBaseURL}
	if s.zesimo != nil && s.zesimo.Configured() {
		wallet, e := s.zesimo.Wallet(r.Context())
		if e != nil {
			writeError(w, http.StatusBadGateway, "zesimo_error", e.Error(), nil)
			return
		}
		response["wallet"] = wallet
	}
	writeJSON(w, 200, response)
}
func (s *Server) syncZesimo(w http.ResponseWriter, r *http.Request) {
	result, e := s.store.SyncZesimoCatalog(r.Context(), s.zesimo)
	if e != nil {
		writeError(w, http.StatusBadGateway, "zesimo_error", e.Error(), nil)
		return
	}
	s.audit(r, "sync", "supplier", result.SupplierID, nil, result)
	writeJSON(w, 200, result)
}

func (s *Server) listEmployees(w http.ResponseWriter, r *http.Request) {
	p, l := pageParams(r)
	q := r.URL.Query()
	x, m, e := s.store.ListEmployees(r.Context(), q.Get("q"), q.Get("role"), q.Get("status"), p, l)
	if e != nil {
		respondStoreError(w, e)
		return
	}
	listResponse(w, x, m)
}
func (s *Server) getEmployee(w http.ResponseWriter, r *http.Request) {
	x, e := s.store.UserByID(r.Context(), r.PathValue("id"))
	if e != nil {
		respondStoreError(w, e)
		return
	}
	writeJSON(w, 200, x)
}
func (s *Server) createEmployee(w http.ResponseWriter, r *http.Request) {
	var in EmployeeInput
	if !decodeJSON(w, r, &in) {
		return
	}
	if in.Name == "" || in.Email == "" || in.Login == "" {
		writeError(w, 422, "validation_error", "name, email and login are required", nil)
		return
	}
	x, e := s.store.CreateEmployee(r.Context(), in)
	if e != nil {
		respondStoreError(w, e)
		return
	}
	s.audit(r, "create", "employee", x.ID, nil, x)
	writeJSON(w, 201, x)
}
func (s *Server) updateEmployee(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	old, e := s.store.UserByID(r.Context(), id)
	if e != nil {
		respondStoreError(w, e)
		return
	}
	var in EmployeeInput
	if !decodeJSON(w, r, &in) {
		return
	}
	x, e := s.store.UpdateEmployee(r.Context(), id, in)
	if e != nil {
		respondStoreError(w, e)
		return
	}
	s.audit(r, "update", "employee", id, old, x)
	writeJSON(w, 200, x)
}
func (s *Server) deleteEmployee(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == currentUser(r).ID {
		writeError(w, 409, "conflict", "You cannot disable your own account", nil)
		return
	}
	old, e := s.store.UserByID(r.Context(), id)
	if e != nil {
		respondStoreError(w, e)
		return
	}
	if e = s.store.DeleteEmployee(r.Context(), id); e != nil {
		respondStoreError(w, e)
		return
	}
	s.audit(r, "disable", "employee", id, old, nil)
	w.WriteHeader(204)
}

func (s *Server) listNotifications(w http.ResponseWriter, r *http.Request) {
	p, l := pageParams(r)
	unread, _ := strconv.ParseBool(r.URL.Query().Get("unread"))
	x, m, e := s.store.ListNotifications(r.Context(), unread, p, l)
	if e != nil {
		respondStoreError(w, e)
		return
	}
	listResponse(w, x, m)
}
func (s *Server) readNotification(w http.ResponseWriter, r *http.Request) {
	if e := s.store.ReadNotification(r.Context(), r.PathValue("id")); e != nil {
		respondStoreError(w, e)
		return
	}
	w.WriteHeader(204)
}
func (s *Server) readAllNotifications(w http.ResponseWriter, r *http.Request) {
	if e := s.store.ReadAllNotifications(r.Context()); e != nil {
		respondStoreError(w, e)
		return
	}
	w.WriteHeader(204)
}
func (s *Server) listAudit(w http.ResponseWriter, r *http.Request) {
	p, l := pageParams(r)
	q := r.URL.Query()
	x, m, e := s.store.ListAudit(r.Context(), q.Get("entityType"), q.Get("actorId"), p, l)
	if e != nil {
		respondStoreError(w, e)
		return
	}
	listResponse(w, x, m)
}

func (s *Server) salesCSV(w http.ResponseWriter, r *http.Request) {
	from, to, e := dateRange(r)
	if e != nil {
		writeError(w, 422, "validation_error", e.Error(), nil)
		return
	}
	rows, e := s.db.Query(r.Context(), `SELECT o.order_number,o.created_at,c.name,s.name,o.country,o.plan_name,o.price,o.cost,o.fee,o.tax,o.profit,o.payment_status,o.status FROM orders o JOIN customers c ON c.id=o.customer_id JOIN suppliers s ON s.id=o.supplier_id WHERE o.created_at>=$1 AND o.created_at<$2 ORDER BY o.created_at`, from, to)
	if e != nil {
		respondStoreError(w, e)
		return
	}
	defer rows.Close()
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="sales-%s-%s.csv"`, from.Format("2006-01-02"), to.AddDate(0, 0, -1).Format("2006-01-02")))
	_, _ = w.Write([]byte{0xEF, 0xBB, 0xBF})
	writer := csv.NewWriter(w)
	writer.Comma = ';'
	_ = writer.Write([]string{"Order", "Created", "Customer", "Supplier", "Country", "Plan", "Price", "Cost", "Fee", "Tax", "Profit", "Payment", "Status"})
	for rows.Next() {
		var number int64
		var created time.Time
		var customer, supplier, country, plan, payment, status string
		var price, cost, fee, tax, profit float64
		if e = rows.Scan(&number, &created, &customer, &supplier, &country, &plan, &price, &cost, &fee, &tax, &profit, &payment, &status); e != nil {
			return
		}
		_ = writer.Write([]string{fmt.Sprintf("#%d", number), created.Format(time.RFC3339), customer, supplier, country, plan, fmt.Sprint(price), fmt.Sprint(cost), fmt.Sprint(fee), fmt.Sprint(tax), fmt.Sprint(profit), payment, status})
	}
	writer.Flush()
}
