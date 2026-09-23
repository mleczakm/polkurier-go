package polkurier

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"
)

// ShipmentType is the kind of consignment (SHIPMENT_TYPE).
type ShipmentType string

// Shipment types.
const (
	ShipmentEnvelope ShipmentType = "envelope"
	ShipmentBox      ShipmentType = "box"
	ShipmentPalette  ShipmentType = "palette"
)

// PackType is the parcel format (PACK_TYPE).
type PackType string

// Pack types.
const (
	PackStandard    PackType = "ST"
	PackNonStandard PackType = "NST"
	PackHalfPallet  PackType = "PPAL"
	PackPallet      PackType = "PAL"
	PackLong        PackType = "DLU"
)

// CODType is when cash-on-delivery money is returned (COD_TYPE).
type CODType string

// COD return times.
const (
	CODStandard CODType = "S"   // 5–7 working days
	COD1Day     CODType = "1D"  // 1 working day
	COD4Days    CODType = "4D"  // 4 working days
	COD16Days   CODType = "16D" // 16 working days
)

// ReturnCOD is how cash-on-delivery money is returned (return_cod).
type ReturnCOD string

// COD return methods.
const (
	ReturnBankAccount ReturnCOD = "BA" // bank transfer
	ReturnPostal      ReturnCOD = "PO" // postal order to the sender
	ReturnMoneyBox    ReturnCOD = "MB" // money box at polkurier.pl
)

// OrderStatus is a shipment status code (ORDER_STATUS).
type OrderStatus string

// Order statuses.
const (
	StatusAwaiting    OrderStatus = "O"  // saved, awaiting payment
	StatusConfirmed   OrderStatus = "P"  // waybill generated, waiting for the courier
	StatusCancelled   OrderStatus = "A"  // cancelled
	StatusInTransit   OrderStatus = "WP" // picked up, on the way
	StatusDelivered   OrderStatus = "D"  // delivered
	StatusReturned    OrderStatus = "Z"  // returned to sender
	StatusException   OrderStatus = "W"  // delivery problem
	StatusMultiPickup OrderStatus = "PZ" // filter only: orders waiting for a collective pickup
)

// Service is an additional service code (ADDITIONAL_SERVICE).
type Service string

// Additional services.
const (
	ServiceReturnDocuments            Service = "ROD"
	ServiceCourierWithLabel           Service = "COURIER_WITH_LABEL"
	ServiceWeekendDelivery            Service = "WEEK_COLLECTION"
	ServiceSMSNotification            Service = "SMS_NOTIFICATION_RECIPIENT"
	ServiceSMSNotificationWithName    Service = "SMS_NOTIFICATION_RECIPIENT_WITH_NAME"
	ServiceLabelless                  Service = "LABELLESS"
	ServiceHandleWithCare             Service = "HANDLE_WITH_CARE"
	ServiceDeliveryToOwnHands         Service = "DOSTAWA_DO_RAK_WLASNYCH"
	ServiceCheckContent               Service = "CHECK_CONTENT"
	ServicePhoneNotification          Service = "PHONE_NOTIFICATION_RECIPIENT"
	ServiceCoverAddressSender         Service = "COVER_ADDRESS_SENDER"
	ServiceTires                      Service = "TIRES"
	ServiceBringingDeliveredParcel    Service = "BRINGING_DELIVERED_PARCEL"
	ServiceSaturdayDelivery           Service = "SATURDAY_DELIVERY"
	ServicePhoneNotificationCollected Service = "PHONE_NOTIFICATION_COLLECTION"
	ServiceDeliveryToTime             Service = "DELIVERY_TO_TIME"
	ServiceHandDelivery               Service = "HAND_DELIVERY"
)

// PointFunction is a capability of a courier point / parcel locker (COURIER_POINT_FUNCTION).
type PointFunction string

// Point functions.
const (
	PointCOD     PointFunction = "cod"
	PointSend    PointFunction = "send"
	PointCollect PointFunction = "collect"
)

// Common courier codes. The authoritative list is returned by [Client.AvailableCarriers].
const (
	CourierInPostLocker = "INPOST_PACZKOMAT"
	CourierInPost       = "INPOST"
	CourierDPD          = "DPD"
	CourierUPS          = "UPS"
	CourierDHL          = "DHL"
)

// Float decodes JSON numbers and numeric strings ("12.30", "12,30"), which the API mixes.
type Float float64

// UnmarshalJSON implements json.Unmarshaler.
func (f *Float) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	if s == "" || s == "null" {
		*f = 0
		return nil
	}
	v, err := strconv.ParseFloat(strings.ReplaceAll(s, ",", "."), 64)
	if err != nil {
		return err
	}
	*f = Float(v)
	return nil
}

// Grosz converts a PLN amount to integer grosz (hundredths), rounding to the nearest.
func (f Float) Grosz() int64 {
	if f < 0 {
		return -int64(-float64(f)*100 + 0.5)
	}
	return int64(float64(f)*100 + 0.5)
}

// Bool decodes JSON booleans as well as 0/1 and "true"/"false" strings.
type Bool bool

// UnmarshalJSON implements json.Unmarshaler.
func (v *Bool) UnmarshalJSON(b []byte) error {
	switch strings.ToLower(strings.Trim(string(b), `"`)) {
	case "true", "1", "yes", "t":
		*v = true
	default:
		*v = false
	}
	return nil
}

// String decodes JSON strings and numbers (IDs are sometimes numeric).
type String string

// UnmarshalJSON implements json.Unmarshaler.
func (v *String) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		*v = ""
		return nil
	}
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		*v = String(s)
		return nil
	}
	*v = String(strings.TrimSpace(string(b)))
	return nil
}

// Date is a calendar day in YYYY-MM-DD form.
type Date string

// NewDate formats t as an API date.
func NewDate(t time.Time) Date { return Date(t.Format("2006-01-02")) }

// Pack is one parcel (ORDER_PACK). Dimensions in centimetres, weight in kilograms.
type Pack struct {
	Length int      `json:"length"`
	Width  int      `json:"width"`
	Height int      `json:"height"`
	Weight float64  `json:"weight"`
	Amount int      `json:"amount,omitempty"` // default 1, at most 99
	Type   PackType `json:"type,omitempty"`   // default ST
}

// Address is a sender or recipient (ORDER_SENDER / ORDER_RECIPIENT). PointID selects a drop-off point for the
// sender or a pick-up point / parcel locker for the recipient.
type Address struct {
	Company     string `json:"company,omitempty"`
	Person      string `json:"person"`
	Street      string `json:"street"`
	HouseNumber string `json:"housenumber"`
	FlatNumber  string `json:"flatnumber,omitempty"`
	Postcode    string `json:"postcode"`
	City        string `json:"city"`
	Email       string `json:"email"`
	Phone       string `json:"phone"`
	Country     string `json:"country,omitempty"` // ISO alpha-2, default PL
	PointID     string `json:"point_id,omitempty"`
}

// CoverAddress replaces the real sender address printed on the label (service COVER_ADDRESS_SENDER).
type CoverAddress struct {
	Company     string `json:"company,omitempty"`
	Person      string `json:"person,omitempty"`
	Street      string `json:"street,omitempty"`
	HouseNumber string `json:"housenumber,omitempty"`
	FlatNumber  string `json:"flatnumber,omitempty"`
	Postcode    string `json:"postcode,omitempty"`
	City        string `json:"city,omitempty"`
	Email       string `json:"email,omitempty"`
	Phone       string `json:"phone,omitempty"`
}

// Pickup is the courier pickup window (ORDER_PICKUP). With NoCourierOrder the parcel is dropped off at a point
// (or the courier is ordered by phone) and the date/time fields are ignored.
type Pickup struct {
	Date           Date   `json:"pickupdate,omitempty"`
	TimeFrom       string `json:"pickuptimefrom,omitempty"` // HH:MM
	TimeTo         string `json:"pickuptimeto,omitempty"`
	NoCourierOrder bool   `json:"nocourierorder"`
	MultiPickup    bool   `json:"multiPickup,omitempty"`
}

// COD is cash on delivery.
type COD struct {
	Type        CODType   `json:"codtype,omitempty"`
	Amount      float64   `json:"codamount,omitempty"`
	BankAccount string    `json:"codbankaccount,omitempty"`
	Return      ReturnCOD `json:"return_cod,omitempty"`
}

// OrderRequest describes a shipment (order_request) for [Client.CreateOrder] and [Client.OrderValuationV2].
type OrderRequest struct {
	ShipmentType     ShipmentType      `json:"shipmenttype"`
	Courier          string            `json:"courier,omitempty"`
	Services         map[Service]bool  `json:"courierservice,omitempty"`
	Description      string            `json:"description,omitempty"` // content, max 30 characters
	Sender           *Address          `json:"sender,omitempty"`
	Recipient        *Address          `json:"recipient,omitempty"`
	CoverAddress     *CoverAddress     `json:"cover_address,omitempty"`
	Packs            []Pack            `json:"packs"`
	Pickup           *Pickup           `json:"pickup,omitempty"`
	COD              *COD              `json:"COD,omitempty"`
	Insurance        float64           `json:"insurance,omitempty"` // PLN
	AdditionalFields map[string]string `json:"additional_fields,omitempty"`
}

// ValuationRequest asks for prices. With ReturnValuations set, only that courier is priced.
// For valuation every field except ShipmentType is optional; the more is given, the more precise the price.
type ValuationRequest struct {
	OrderRequest
	ReturnValuations string `json:"returnvaluations,omitempty"`
}

// Valuation is the price of a shipment with one courier.
type Valuation struct {
	ServiceCode           string `json:"servicecode"`
	ServiceName           string `json:"-"`
	NetPrice              Float  `json:"netprice"`
	GrossPrice            Float  `json:"grossprice"`
	ConditionalPriceNet   Float  `json:"conditional_price_nett"`
	ConditionalPriceGross Float  `json:"conditional_price_gross"`
	PromotionNet          Float  `json:"promotion_nett"`
	PromotionGross        Float  `json:"promotion_gross"`
	RebateNet             Float  `json:"rebate_nett"`
	RebateGross           Float  `json:"rebate_gross"`
	Shipment              Bool   `json:"shipment"`
	Available             Bool   `json:"available"`
	UnavailableMessage    string `json:"unavailable_message"`
}

// UnmarshalJSON accepts both "servicename" (v1) and "serviceName" (v2).
func (v *Valuation) UnmarshalJSON(b []byte) error {
	type plain Valuation
	var p struct {
		plain
		Name1 string `json:"servicename"`
		Name2 string `json:"serviceName"`
	}
	if err := json.Unmarshal(b, &p); err != nil {
		return err
	}
	*v = Valuation(p.plain)
	v.ServiceName = p.Name1
	if v.ServiceName == "" {
		v.ServiceName = p.Name2
	}
	return nil
}

// FinalGross is the gross price to be paid: the promotional price when there is one, otherwise the regular one.
func (v Valuation) FinalGross() Float {
	if v.PromotionGross > 0 && v.PromotionGross < v.GrossPrice {
		return v.PromotionGross
	}
	return v.GrossPrice
}

// CreatedOrder is the result of [Client.CreateOrder].
type CreatedOrder struct {
	OrderNumber  string   `json:"order_number"`
	Waybills     []string `json:"label"`
	PriceGross   Float    `json:"price_gross"`
	PriceNet     Float    `json:"price_net"`
	TrackingURL  string   `json:"url_tracktrace"`
	UnpaidAmount Float    `json:"unpaid_amount"`
	IsPaid       Bool     `json:"is_paid"`
}

// Status is the tracking status of a shipment.
type Status struct {
	URL           string      `json:"url"`
	StatusDate    string      `json:"status_date"`
	Status        string      `json:"status"`
	StatusCode    OrderStatus `json:"status_code"`
	DeliveredDate string      `json:"delivered_date"`
}

// Feature describes an optional capability of a carrier (available_carriers additional_data).
type Feature struct {
	Available   Bool   `json:"available"`
	Required    Bool   `json:"required"`
	Status      Bool   `json:"status"`
	Description string `json:"description"`
}

// FieldOption is a choice of a SELECT additional field.
type FieldOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// AdditionalField is a carrier-specific order parameter for OrderRequest.AdditionalFields.
type AdditionalField struct {
	Name        string        `json:"name"`
	Label       string        `json:"label"`
	Description string        `json:"description"`
	Type        string        `json:"type"` // SELECT or TEXT
	Options     []FieldOption `json:"options"`
	Required    Bool          `json:"required"`
}

// CarrierDetails are returned when AvailableCarriers is asked for additional data.
type CarrierDetails struct {
	COD              Feature                  `json:"COD"`
	Insurance        Feature                  `json:"insurance"`
	ShipmentTypes    map[ShipmentType]Feature `json:"shipmenttype"`
	Services         map[Service]Feature      `json:"courierservice"`
	Pickup           map[string]Feature       `json:"pickup"`
	AdditionalFields []AdditionalField        `json:"additional_fields"`
}

// Carrier is a courier service offered by polkurier.pl.
type Carrier struct {
	ServiceCode      string          `json:"servicecode"`
	Name             string          `json:"name"`
	ForeignShipments Bool            `json:"foreign_shipments"`
	Details          *CarrierDetails `json:"additional_data,omitempty"`
}

// PickupWindow is an hour range for a courier pickup.
type PickupWindow struct {
	From string `json:"timefrom"`
	To   string `json:"timeto"`
}

// PickupDay lists the pickup windows of one day.
type PickupDay struct {
	Date    Date           `json:"pickupdate"`
	Windows []PickupWindow `json:"time"`
}

// PickupAvailability answers [Client.PickupCourier].
type PickupAvailability struct {
	Available Bool           `json:"pickupdate"`
	Windows   []PickupWindow `json:"time"`
}

// Country is a destination country.
type Country struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	IsEU      Bool       `json:"is_ue"`
	IsDefault Bool       `json:"is_default"`
	Provinces []Province `json:"provinces"`
}

// Province is an administrative region (e.g. a US state).
type Province struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// CourierPoint is a parcel locker, pickup point or post office.
type CourierPoint struct {
	ID                String          `json:"id"`
	Type              string          `json:"type"`
	Name              string          `json:"name"`
	Provider          string          `json:"provider"`
	City              string          `json:"city"`
	Zip               string          `json:"zip"`
	Street            string          `json:"street"`
	Address           string          `json:"address"`
	Description       string          `json:"description"`
	Latitude          Float           `json:"latitude"`
	Longitude         Float           `json:"longitude"`
	COD               Bool            `json:"cod"`
	Available         Bool            `json:"available"`
	Status            string          `json:"status"`
	Send              Bool            `json:"send"`
	Collect           Bool            `json:"collect"`
	OpeningHours      string          `json:"openingHours"`
	Visible           Bool            `json:"visible"`
	RequireApp        Bool            `json:"requireApp"`
	RequireAppMessage string          `json:"requireAppMessage"`
	Functions         []PointFunction `json:"functions"`
	CountryISO        string          `json:"countryiso"`
}

// PointQuery filters [Client.GetCourierPoints].
type PointQuery struct {
	Couriers    []string        `json:"couriers"`
	ID          string          `json:"id,omitempty"`
	SearchQuery string          `json:"searchquery,omitempty"`
	Functions   []PointFunction `json:"functions,omitempty"`
	Limit       int             `json:"limit,omitempty"`
	Page        int             `json:"page,omitempty"`
}

// Waybill is a consignment note of an order.
type Waybill struct {
	Number    string `json:"number"`
	IsDefault Bool   `json:"is_default"`
}

// OrderItem is a priced line of an order.
type OrderItem struct {
	Description string `json:"description"`
	Amount      Float  `json:"amount"`
	Price       Float  `json:"price"`
	ValueNet    Float  `json:"value_nett"`
	ValueGross  Float  `json:"value_gross"`
	Type        string `json:"type"` // SERVICE or PACK
}

// CODReturn is an operation on the COD amount (payout, compensation, fee).
type CODReturn struct {
	Reason         string   `json:"reason"`
	Description    string   `json:"description"`
	DocumentNumber string   `json:"document_number"`
	InvoiceNumbers []string `json:"invoice_number"`
	Date           string   `json:"date"`
	Amount         Float    `json:"amount"`
}

// OrderCOD is the cash-on-delivery part of an order.
type OrderCOD struct {
	Type          CODType     `json:"codtype"`
	Amount        Float       `json:"codamount"`
	BankAccount   String      `json:"codbankaccount"`
	Return        ReturnCOD   `json:"return_cod"`
	Status        string      `json:"status"`
	ReturnDetails []CODReturn `json:"return_details"`
}

// OrderPack is a parcel of an existing order.
type OrderPack struct {
	Length     Float    `json:"length"`
	Width      Float    `json:"width"`
	Height     Float    `json:"height"`
	Weight     Float    `json:"weight"`
	Amount     Float    `json:"amount"`
	Type       PackType `json:"type"`
	PriceNet   Float    `json:"price_net"`
	PriceGross Float    `json:"price_gross"`
}

// OrderPickup is the pickup of an existing order.
type OrderPickup struct {
	Date           *string `json:"pickupdate"`
	TimeFrom       *string `json:"pickuptimefrom"`
	TimeTo         *string `json:"pickuptimeto"`
	NoCourierOrder Bool    `json:"nocourierorder"`
}

// OrderParty is a sender or recipient of an existing order.
type OrderParty struct {
	Company  string `json:"company"`
	Person   string `json:"person"`
	Street   string `json:"street"`
	Postcode string `json:"postcode"`
	City     string `json:"city"`
	Phone    string `json:"phone"`
	Email    string `json:"email"`
	Country  string `json:"country"`
	PointID  string `json:"point_id"`
}

// Order is an order placed at polkurier.pl (get_orders).
type Order struct {
	Number        string       `json:"number"`
	Date          string       `json:"date"`
	Description   string       `json:"description"`
	Courier       string       `json:"courier"`
	CourierName   string       `json:"courier_name"`
	ShipmentType  ShipmentType `json:"shipmenttype"`
	Status        string       `json:"status"`
	StatusCode    OrderStatus  `json:"status_code"`
	PriceGross    Float        `json:"price_gross"`
	PriceNet      Float        `json:"price_net"`
	IsPaid        Bool         `json:"is_paid"`
	HasInvoice    Bool         `json:"has_invoice"`
	UnpaidAmount  Float        `json:"unpaid_amount"`
	URL           string       `json:"url"`
	StatusDate    string       `json:"status_date"`
	Waybills      []Waybill    `json:"waybills"`
	DeliveredDate string       `json:"delivered_date"`
	Insurance     Float        `json:"insurance"`
	Sender        OrderParty   `json:"sender"`
	Recipient     OrderParty   `json:"recipient"`
	COD           *OrderCOD    `json:"COD"`
	Pickup        *OrderPickup `json:"pickup"`
	Packs         []OrderPack  `json:"packs"`
	Items         []OrderItem  `json:"items"`
}

// OrdersQuery filters [Client.GetOrders].
type OrdersQuery struct {
	Status   OrderStatus `json:"status,omitempty"`
	Courier  string      `json:"courier,omitempty"`
	CODMin   string      `json:"cod_min,omitempty"`
	CODMax   string      `json:"cod_max,omitempty"`
	DateFrom Date        `json:"date_from,omitempty"`
	DateTo   Date        `json:"date_to,omitempty"`
	Search   string      `json:"search,omitempty"`
	PageSize int         `json:"pagesize,omitempty"` // max 100
	Page     int         `json:"page,omitempty"`
	Packs    bool        `json:"packs,omitempty"`
	Items    bool        `json:"items,omitempty"`
	Files    bool        `json:"files,omitempty"`
}

// OrdersPage is a page of orders.
type OrdersPage struct {
	TotalRows   int     `json:"totalrows"`
	TotalPages  int     `json:"totalpages"`
	CurrentPage int     `json:"currentpage"`
	PageSize    int     `json:"pagesize"`
	Orders      []Order `json:"result"`
}

// MultiPickupResult reports which orders got a collective courier pickup.
type MultiPickupResult struct {
	Success []string          `json:"success"`
	Failed  map[string]string `json:"failed"`
}

// AddressBookType distinguishes sender and recipient addresses.
type AddressBookType string

// Address book types.
const (
	AddressSender    AddressBookType = "SENDER"
	AddressRecipient AddressBookType = "RECIPIENT"
)

// AddressBookEntry is an address saved in the polkurier.pl address book.
type AddressBookEntry struct {
	ID             String          `json:"id,omitempty"`
	Name           string          `json:"name"`
	Company        string          `json:"company"`
	Person         string          `json:"person"`
	City           string          `json:"city"`
	Postcode       string          `json:"postcode"`
	Street         string          `json:"street"`
	HouseNumber    string          `json:"housenumber"`
	FlatNumber     string          `json:"flatnumber"`
	Email          string          `json:"email"`
	Phone          string          `json:"phone"`
	Country        string          `json:"country"`
	Province       string          `json:"province"`
	Type           AddressBookType `json:"type"`
	Default        Bool            `json:"default"`
	IsCoverAddress Bool            `json:"is_cover_address"`
}

// BankAccount is an account for COD payouts.
type BankAccount struct {
	ID      String `json:"id,omitempty"`
	Number  string `json:"number"`
	Name    string `json:"name"`
	Default Bool   `json:"default"`
}

// PackTemplate is a saved parcel template.
type PackTemplate struct {
	ID           String       `json:"id,omitempty"`
	Name         string       `json:"name"`
	ShipmentType ShipmentType `json:"shipmenttype"`
	Length       Float        `json:"length"`
	Width        Float        `json:"width"`
	Height       Float        `json:"height"`
	Weight       Float        `json:"weight"`
	COD          Float        `json:"COD"`
	Insurance    Float        `json:"insurance"`
	Description  string       `json:"description"`
	Type         PackType     `json:"type"`
	Courier      *string      `json:"courier"`
	Services     []Service    `json:"services"`
}

// TwoFactorSession is a two-factor verification session (SMS code) for sensitive operations.
type TwoFactorSession struct {
	ID              string `json:"id"`
	PhoneNumber     string `json:"phone_number"`
	OperationTitle  string `json:"operation_title"`
	OperationNumber int    `json:"operation_number"`
	OperationDate   string `json:"operation_date"`
	SentAt          string `json:"sent_at"`
	ExpiresAt       string `json:"expires_at"`
}

// Two-factor namespaces.
const TwoFactorChangeBankAccount = "CHANGE_BANK_ACCOUNT_NUMBER"
