package polkurier

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// fakeAPI answers every call with the given response for the given method and records the last request.
type fakeAPI struct {
	t        *testing.T
	last     map[string]any
	answers  map[string]string
	httpCode int
}

func (f *fakeAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	if r.Method != http.MethodPost || r.Header.Get("Content-Type") != "application/json" {
		f.t.Errorf("unexpected request %s %s", r.Method, r.Header.Get("Content-Type"))
	}
	f.last = map[string]any{}
	if err := json.Unmarshal(body, &f.last); err != nil {
		f.t.Fatal(err)
	}
	if f.httpCode != 0 {
		w.WriteHeader(f.httpCode)
		return
	}
	method, _ := f.last["apimethod"].(string)
	answer, ok := f.answers[method]
	if !ok {
		answer = `{"status":"error","response":"unknown method ` + method + `"}`
	}
	_, _ = io.WriteString(w, answer)
}

func newTest(t *testing.T, answers map[string]string) (*Client, *fakeAPI) {
	f := &fakeAPI{t: t, answers: answers}
	srv := httptest.NewServer(f)
	t.Cleanup(srv.Close)
	return New("12345", "secret", WithBaseURL(srv.URL), WithPlatform("ceramiza", "1.0")), f
}

func data(f *fakeAPI) map[string]any { return f.last["data"].(map[string]any) }

func TestEnvelopeAndAuth(t *testing.T) {
	c, f := newTest(t, map[string]string{"test_auth_api": `{"status":"success","response":{"authorization":true}}`})
	ok, err := c.TestAuth(context.Background())
	if err != nil || !ok {
		t.Fatalf("TestAuth = %v, %v", ok, err)
	}
	auth := f.last["authorization"].(map[string]any)
	if auth["login"] != "12345" || auth["token"] != "secret" || f.last["apimethod"] != "test_auth_api" {
		t.Fatalf("bad envelope: %v", f.last)
	}
	if d := data(f); d["platform"] != "ceramiza" || d["platform_version"] != "1.0" {
		t.Fatalf("platform not sent: %v", d)
	}
}

func TestAPIError(t *testing.T) {
	c, _ := newTest(t, map[string]string{"cancel_order": `{"status":"error","response":"Zamówienie 1234-1 zostało już anulowane"}`})
	_, err := c.CancelOrder(context.Background(), "1234-1")
	if !IsAPIError(err) || err.Error() != "polkurier cancel_order: Zamówienie 1234-1 zostało już anulowane" {
		t.Fatalf("unexpected error %v", err)
	}
}

func TestHTTPError(t *testing.T) {
	c, f := newTest(t, nil)
	f.httpCode = http.StatusBadGateway
	if _, err := c.Heartbeat(context.Background()); err == nil || IsAPIError(err) {
		t.Fatalf("expected transport error, got %v", err)
	}
}

func TestValuationV2(t *testing.T) {
	c, f := newTest(t, map[string]string{"order_valuation_v2": `{"status":"success","response":[
		{"servicecode":"INPOST_PACZKOMAT","serviceName":"InPost Paczkomat","netprice":"12,19","grossprice":14.99,
		 "promotion_gross":13.49,"shipment":true,"available":1,"unavailable_message":""},
		{"servicecode":"DPD","servicename":"DPD Classic","netprice":16.25,"grossprice":"19.99","available":false,
		 "unavailable_message":"Brak"}]}`})
	vals, err := c.OrderValuationV2(context.Background(), ValuationRequest{
		OrderRequest: OrderRequest{
			ShipmentType: ShipmentBox,
			Packs:        []Pack{{Length: 30, Width: 20, Height: 20, Weight: 2}},
			Sender:       &Address{Postcode: "05-250"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(vals) != 2 || vals[0].ServiceName != "InPost Paczkomat" || vals[1].ServiceName != "DPD Classic" {
		t.Fatalf("names: %+v", vals)
	}
	if vals[0].NetPrice.Grosz() != 1219 || vals[0].FinalGross().Grosz() != 1349 || !bool(vals[0].Available) {
		t.Fatalf("prices: %+v", vals[0])
	}
	if vals[1].GrossPrice.Grosz() != 1999 || bool(vals[1].Available) {
		t.Fatalf("second: %+v", vals[1])
	}
	d := data(f)
	if d["shipmenttype"] != "box" || d["sender"].(map[string]any)["postcode"] != "05-250" {
		t.Fatalf("request data: %v", d)
	}
	if _, has := d["recipient"]; has {
		t.Fatal("nil recipient must be omitted")
	}
}

func TestCreateOrderAndLabel(t *testing.T) {
	pdf := []byte("%PDF-1.4 test")
	c, f := newTest(t, map[string]string{
		"create_order": `{"status":"success","response":{"order_number":"1234-10","label":["13299300045383"],
			"price_gross":14.99,"price_net":12.19,"url_tracktrace":"https://inpost.pl/?n=1","unpaid_amount":0,"is_paid":true}}`,
		"get_label": `{"status":"success","response":{"file":"` + base64.StdEncoding.EncodeToString(pdf) + `"}}`,
	})
	ctx := context.Background()
	o, err := c.CreateOrder(ctx, OrderRequest{
		ShipmentType: ShipmentBox, Courier: CourierInPostLocker, Description: "Ceramiczna lampka ręcznie robiona — delikatne!",
		Sender:    &Address{Person: "Iza", Street: "Gliniana", HouseNumber: "1", Postcode: "05-250", City: "Radzymin", Email: "a@b.pl", Phone: "500600700"},
		Recipient: &Address{Person: "Anna", Postcode: "00-001", City: "Warszawa", Email: "c@d.pl", Phone: "600700800", PointID: "WAW01M"},
		Packs:     []Pack{{Length: 30, Width: 20, Height: 20, Weight: 2}},
		Pickup:    &Pickup{NoCourierOrder: true},
		Services:  map[Service]bool{ServiceHandleWithCare: true},
		Insurance: 250,
	})
	if err != nil {
		t.Fatal(err)
	}
	if o.OrderNumber != "1234-10" || o.Waybills[0] != "13299300045383" || !bool(o.IsPaid) || o.PriceGross.Grosz() != 1499 {
		t.Fatalf("order: %+v", o)
	}
	d := data(f)
	if n := len([]rune(d["description"].(string))); n != 30 {
		t.Fatalf("description must be cut to 30 characters, got %d", n)
	}
	if d["recipient"].(map[string]any)["point_id"] != "WAW01M" || d["pickup"].(map[string]any)["nocourierorder"] != true {
		t.Fatalf("request: %v", d)
	}
	if d["courierservice"].(map[string]any)["HANDLE_WITH_CARE"] != true {
		t.Fatalf("services: %v", d["courierservice"])
	}
	got, err := c.GetLabel(ctx, "1234-10")
	if err != nil || string(got) != string(pdf) {
		t.Fatalf("label: %q %v", got, err)
	}
	if nums := data(f)["orderno"].([]any); nums[0] != "1234-10" {
		t.Fatalf("orderno: %v", nums)
	}
}

func TestCourierPointsAndStatus(t *testing.T) {
	c, f := newTest(t, map[string]string{
		"get_courier_point": `{"status":"success","response":[{"id":"ADA01M","type":"parcel_locker","provider":"INPOST_PACZKOMAT",
			"city":"Adamów","zip":"21-412","street":"Kościuszki 27","latitude":51.73834,"longitude":"22.26405","cod":true,
			"available":true,"send":true,"collect":true,"functions":["cod","collect","send"]}]}`,
		"get_status": `{"status":"success","response":{"url":"https://x","status":"Dostarczone","status_code":"D","delivered_date":"2018-06-12"}}`,
	})
	ctx := context.Background()
	pts, err := c.GetCourierPoints(ctx, PointQuery{Couriers: []string{CourierInPostLocker}, SearchQuery: "Adamów", Limit: 10})
	if err != nil || len(pts) != 1 || pts[0].ID != "ADA01M" || pts[0].Longitude != 22.26405 || !bool(pts[0].Collect) {
		t.Fatalf("points: %+v %v", pts, err)
	}
	if d := data(f); d["searchquery"] != "Adamów" || d["limit"] != float64(10) {
		t.Fatalf("query: %v", d)
	}
	if _, err := c.GetCourierPoints(ctx, PointQuery{}); err == nil {
		t.Fatal("couriers are required")
	}
	st, err := c.GetStatus(ctx, "1234-1")
	if err != nil || st.StatusCode != StatusDelivered {
		t.Fatalf("status: %+v %v", st, err)
	}
}

func TestCarriersWithDetails(t *testing.T) {
	c, _ := newTest(t, map[string]string{"available_carriers": `{"status":"sukces","response":[]}`})
	if _, err := c.AvailableCarriers(context.Background(), true, ""); !IsAPIError(err) {
		t.Fatal(`only "success" is a successful status`)
	}
	c, _ = newTest(t, map[string]string{"available_carriers": `{"status":"success","response":[{"servicecode":"UPS","name":"UPS - Standard",
		"foreign_shipments":false,"additional_data":{"COD":{"available":true,"description":"Pobranie"},
		"shipmenttype":{"box":{"available":true,"description":"Paczka"}},"courierservice":{"ROD":{"available":true,"description":"Zwrot dokumentów"}},
		"pickup":{"pickupdate":{"status":true,"description":"data"},"nocourierorder":{"available":true,"description":"tel"}},
		"additional_fields":[{"name":"external_transport_security","label":"Zabezpieczenie","type":"SELECT","options":[{"value":"TAPE","label":"Taśma"}],"required":false}]}}]}`})
	carriers, err := c.AvailableCarriers(context.Background(), true, "UPS")
	if err != nil || len(carriers) != 1 {
		t.Fatalf("carriers: %v %v", carriers, err)
	}
	d := carriers[0].Details
	if !bool(d.COD.Available) || !bool(d.ShipmentTypes[ShipmentBox].Available) || !bool(d.Services[ServiceReturnDocuments].Available) ||
		!bool(d.Pickup["nocourierorder"].Available) || d.AdditionalFields[0].Options[0].Value != "TAPE" {
		t.Fatalf("details: %+v", d)
	}
}

func TestFloatGrosz(t *testing.T) {
	for in, want := range map[string]int64{`14.99`: 1499, `"14,99"`: 1499, `"0.1"`: 10, `null`: 0, `""`: 0, `203.17`: 20317} {
		var f Float
		if err := json.Unmarshal([]byte(in), &f); err != nil || f.Grosz() != want {
			t.Errorf("%s -> %d (%v), want %d", in, f.Grosz(), err, want)
		}
	}
}
