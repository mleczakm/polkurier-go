package polkurier

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"
)

// Heartbeat checks that the API is online and returns the server time.
func (c *Client) Heartbeat(ctx context.Context) (time.Time, error) {
	var out struct {
		Time string `json:"time"`
	}
	if err := c.Call(ctx, "heartbeat", struct{}{}, &out); err != nil {
		return time.Time{}, err
	}
	return time.Parse(time.RFC3339, out.Time)
}

// TestAuth reports whether the login and token are valid.
func (c *Client) TestAuth(ctx context.Context) (bool, error) {
	var out struct {
		Authorization Bool `json:"authorization"`
	}
	err := c.Call(ctx, "test_auth_api", struct{}{}, &out)
	return bool(out.Authorization), err
}

// AvailableCarriers lists couriers. With details, services, shipment types and additional fields are included;
// courier ("servicecode") limits the answer to one carrier.
func (c *Client) AvailableCarriers(ctx context.Context, details bool, courier string) ([]Carrier, error) {
	data := map[string]any{"additional_data": details}
	if courier != "" {
		data["returncarrier"] = courier
	}
	var out []Carrier
	err := c.Call(ctx, "available_carriers", data, &out)
	return out, err
}

// LegacyValuationRequest is the input of the deprecated order_valuation method.
type LegacyValuationRequest struct {
	ReturnValuations  string           `json:"returnvaluations,omitempty"`
	ShipmentType      ShipmentType     `json:"shipmenttype"`
	Packs             []Pack           `json:"packs"`
	COD               float64          `json:"COD,omitempty"`
	CODType           CODType          `json:"codtype,omitempty"`
	ReturnCOD         ReturnCOD        `json:"return_cod,omitempty"`
	Insurance         float64          `json:"insurance,omitempty"`
	PostcodeRecipient string           `json:"postcode_recipient,omitempty"`
	PostcodeSender    string           `json:"postcode_sender,omitempty"`
	RecipientCountry  string           `json:"recipient_country,omitempty"`
	Services          map[Service]bool `json:"courierservice,omitempty"`
}

// OrderValuation prices a shipment with the deprecated order_valuation method.
//
// Deprecated: use [Client.OrderValuationV2].
func (c *Client) OrderValuation(ctx context.Context, req LegacyValuationRequest) ([]Valuation, error) {
	var out []Valuation
	err := c.Call(ctx, "order_valuation", req, &out)
	return out, err
}

// OrderValuationV2 prices a shipment for all couriers (or one, via ReturnValuations). Couriers that cannot carry
// the parcel are omitted; unavailable ones come with Available=false and a message.
func (c *Client) OrderValuationV2(ctx context.Context, req ValuationRequest) ([]Valuation, error) {
	var out []Valuation
	err := c.Call(ctx, "order_valuation_v2", req, &out)
	return out, err
}

// CreateOrder places a shipment order and returns its number, waybill(s) and price.
func (c *Client) CreateOrder(ctx context.Context, req OrderRequest) (*CreatedOrder, error) {
	if req.Description != "" && len([]rune(req.Description)) > 30 {
		req.Description = string([]rune(req.Description)[:30])
	}
	var out CreatedOrder
	if err := c.Call(ctx, "create_order", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) pdf(ctx context.Context, method string, orderNumbers []string) ([]byte, error) {
	if len(orderNumbers) == 0 {
		return nil, fmt.Errorf("polkurier %s: at least one order number is required", method)
	}
	var out struct {
		File string `json:"file"`
	}
	if err := c.Call(ctx, method, map[string]any{"orderno": orderNumbers}, &out); err != nil {
		return nil, err
	}
	pdf, err := base64.StdEncoding.DecodeString(out.File)
	if err != nil {
		return nil, fmt.Errorf("polkurier %s: decode file: %w", method, err)
	}
	return pdf, nil
}

// GetLabel returns the label(s) of the orders as one PDF.
func (c *Client) GetLabel(ctx context.Context, orderNumbers ...string) ([]byte, error) {
	return c.pdf(ctx, "get_label", orderNumbers)
}

// GetProtocol returns the handover protocol of the orders as a PDF.
func (c *Client) GetProtocol(ctx context.Context, orderNumbers ...string) ([]byte, error) {
	return c.pdf(ctx, "get_protocol", orderNumbers)
}

// GetStatus returns the tracking status of an order.
func (c *Client) GetStatus(ctx context.Context, orderNumber string) (*Status, error) {
	var out Status
	if err := c.Call(ctx, "get_status", map[string]string{"orderno": orderNumber}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CancelOrder cancels an order that is still awaiting or confirmed (not yet in transit).
func (c *Client) CancelOrder(ctx context.Context, orderNumber string) (bool, error) {
	var out struct {
		Cancellation Bool `json:"cancellation"`
	}
	err := c.Call(ctx, "cancel_order", map[string]string{"orderno": orderNumber}, &out)
	return bool(out.Cancellation), err
}

// PickupCourier returns the pickup windows of a courier on a date for a sender postcode.
func (c *Client) PickupCourier(ctx context.Context, courier string, date Date, senderPostcode string, shipment ShipmentType) (*PickupAvailability, error) {
	var out PickupAvailability
	err := c.Call(ctx, "pickup_courier", map[string]any{
		"pickupdate": date, "courier": courier, "shipfrom": senderPostcode, "parcel": shipment,
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// GetCourierPickupTime returns pickup days and windows for the next five working days.
func (c *Client) GetCourierPickupTime(ctx context.Context, courier, senderPostcode, recipientPostcode string, shipment ShipmentType) ([]PickupDay, error) {
	data := map[string]any{"courier": courier, "parcel": shipment, "shipmenttype": shipment}
	if senderPostcode != "" {
		data["shipfrom"] = senderPostcode
	}
	if recipientPostcode != "" {
		data["shipto"] = recipientPostcode
	}
	var out []PickupDay
	err := c.Call(ctx, "get_courier_pickup_time", data, &out)
	return out, err
}

// GetCountries returns destination countries (code → name) of an export courier such as UPS_EX.
func (c *Client) GetCountries(ctx context.Context, courier string) (map[string]string, error) {
	out := map[string]string{}
	err := c.Call(ctx, "get_countries", map[string]string{"couriers": courier}, &out)
	return out, err
}

// GetSupportedCountries returns every supported country with provinces.
func (c *Client) GetSupportedCountries(ctx context.Context) ([]Country, error) {
	var out []Country
	err := c.Call(ctx, "get_supported_countries_v2", struct{}{}, &out)
	return out, err
}

// GetCourierPoints searches parcel lockers and pickup points. Use Limit/Page — the full list is large.
func (c *Client) GetCourierPoints(ctx context.Context, q PointQuery) ([]CourierPoint, error) {
	if len(q.Couriers) == 0 {
		return nil, fmt.Errorf("polkurier get_courier_point: at least one courier is required")
	}
	var out []CourierPoint
	err := c.Call(ctx, "get_courier_point", q, &out)
	return out, err
}

// GetOrder returns the details of one order.
func (c *Client) GetOrder(ctx context.Context, orderNumber string) (*Order, error) {
	var out Order
	if err := c.Call(ctx, "get_orders", map[string]string{"orderno": orderNumber}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetOrders lists orders, newest first.
func (c *Client) GetOrders(ctx context.Context, q OrdersQuery) (*OrdersPage, error) {
	var out OrdersPage
	if err := c.Call(ctx, "get_orders", q, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// IsMultiPickupAvailable reports whether a collective pickup can be used for such a shipment.
func (c *Client) IsMultiPickupAvailable(ctx context.Context, courier, senderPostcode string, shipment ShipmentType, packs []Pack) (bool, error) {
	var out struct {
		Available Bool `json:"isMultiPickupAvailable"`
	}
	err := c.Call(ctx, "is_multi_pickup_available", map[string]any{
		"senderPostCode": senderPostcode, "shipmentType": shipment, "courier": courier, "packs": packs,
	}, &out)
	return bool(out.Available), err
}

// CreateMultiOrderPickup orders one courier pickup for orders created with Pickup.MultiPickup.
func (c *Client) CreateMultiOrderPickup(ctx context.Context, date Date, from, to string, orderNumbers []string) (*MultiPickupResult, error) {
	var out MultiPickupResult
	err := c.Call(ctx, "create_multi_order_pickup", map[string]any{
		"pickupdate": date, "pickuptimefrom": from, "pickuptimeto": to, "orders": orderNumbers,
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// GetAddresses returns address book entries, optionally of one type.
func (c *Client) GetAddresses(ctx context.Context, typ AddressBookType) ([]AddressBookEntry, error) {
	data := map[string]any{}
	if typ != "" {
		data["type"] = typ
	}
	var out []AddressBookEntry
	err := c.Call(ctx, "get_addresses", data, &out)
	return out, err
}

// SaveAddress creates (empty ID) or updates an address book entry.
func (c *Client) SaveAddress(ctx context.Context, a AddressBookEntry) (*AddressBookEntry, error) {
	var out AddressBookEntry
	if err := c.Call(ctx, "save_address", a, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteAddress removes an address book entry.
func (c *Client) DeleteAddress(ctx context.Context, typ AddressBookType, id string) error {
	return c.Call(ctx, "delete_address", map[string]any{"type": typ, "id": id}, nil)
}

// GetBankAccounts lists COD payout accounts.
func (c *Client) GetBankAccounts(ctx context.Context) ([]BankAccount, error) {
	var out []BankAccount
	err := c.Call(ctx, "get_bank_accounts", struct{}{}, &out)
	return out, err
}

// SaveBankAccount creates or updates a COD payout account. Adding an account or changing its number or default
// flag needs a two-factor session: see [Client.TwoFactorTokenRegister].
func (c *Client) SaveBankAccount(ctx context.Context, a BankAccount, twoFactorToken, twoFactorCode string) (*BankAccount, error) {
	data := map[string]any{"number": a.Number, "name": a.Name, "default": bool(a.Default)}
	if a.ID != "" {
		data["id"] = a.ID
	}
	if twoFactorToken != "" {
		data["two_factor_token"] = twoFactorToken
		data["two_factor_code"] = twoFactorCode
	}
	var out BankAccount
	if err := c.Call(ctx, "save_bank_account", data, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteBankAccount removes a COD payout account.
func (c *Client) DeleteBankAccount(ctx context.Context, id string) error {
	return c.Call(ctx, "delete_bank_account", map[string]string{"id": id}, nil)
}

// GetPackTemplates lists saved parcel templates.
func (c *Client) GetPackTemplates(ctx context.Context) ([]PackTemplate, error) {
	var out []PackTemplate
	err := c.Call(ctx, "get_pack_templates", struct{}{}, &out)
	return out, err
}

// SavePackTemplate creates (empty ID) or updates a parcel template.
func (c *Client) SavePackTemplate(ctx context.Context, t PackTemplate) (*PackTemplate, error) {
	var out PackTemplate
	if err := c.Call(ctx, "save_pack_template", t, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeletePackTemplate removes a parcel template.
func (c *Client) DeletePackTemplate(ctx context.Context, id string) error {
	return c.Call(ctx, "delete_pack_template", map[string]string{"id": id}, nil)
}

// FindCity returns town names for a postcode.
func (c *Client) FindCity(ctx context.Context, postcode string) ([]string, error) {
	var out []struct {
		Name string `json:"name"`
	}
	if err := c.Call(ctx, "find_city", map[string]string{"postcode": postcode}, &out); err != nil {
		return nil, err
	}
	names := make([]string, 0, len(out))
	for _, o := range out {
		names = append(names, o.Name)
	}
	return names, nil
}

// GetMapToken returns a JWT (valid 4 hours) for the points map widget (maps.polkurier.pl).
func (c *Client) GetMapToken(ctx context.Context) (string, error) {
	var out struct {
		Token string `json:"token"`
	}
	err := c.Call(ctx, "get_map_token", struct{}{}, &out)
	return out.Token, err
}

// TwoFactorTokenRegister starts a two-factor session; a code is sent by SMS to the account phone.
func (c *Client) TwoFactorTokenRegister(ctx context.Context, namespace string) (*TwoFactorSession, error) {
	var out TwoFactorSession
	if err := c.Call(ctx, "two_factor_token_register", map[string]string{"namespace": namespace}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// TwoFactorTokenResend sends the verification code again.
func (c *Client) TwoFactorTokenResend(ctx context.Context, token string) (*TwoFactorSession, error) {
	var out TwoFactorSession
	if err := c.Call(ctx, "two_factor_token_resend", map[string]string{"token": token}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// TwoFactorTokenVerify checks a code before using it (optional).
func (c *Client) TwoFactorTokenVerify(ctx context.Context, token, code string) (bool, error) {
	var out struct {
		Valid Bool `json:"valid"`
	}
	err := c.Call(ctx, "two_factor_token_verify", map[string]string{"token": token, "code": code}, &out)
	return bool(out.Valid), err
}
