# polkurier-go

[![Go Reference](https://pkg.go.dev/badge/github.com/mleczakm/polkurier-go.svg)](https://pkg.go.dev/github.com/mleczakm/polkurier-go)
[![CI](https://github.com/mleczakm/polkurier-go/actions/workflows/ci.yml/badge.svg)](https://github.com/mleczakm/polkurier-go/actions/workflows/ci.yml)

Go client for the [Polkurier](https://www.polkurier.pl) WebService API — price, order and track shipments with
InPost (parcel lockers and courier), DPD, DHL, UPS, Pocztex and the other carriers available at polkurier.pl.

Unofficial; based on the API documentation v1.12 and the official PHP SDK
([Polkurier/polkurier-sdk](https://github.com/Polkurier/polkurier-sdk)). No dependencies outside the standard library.

```sh
go get github.com/mleczakm/polkurier-go
```

## Usage

```go
c := polkurier.New(login, token)               // account ID + API token from the polkurier.pl panel
// polkurier.New(login, token, polkurier.WithSandbox())  // https://api-sandbox.polkurier.pl

box := []polkurier.Pack{{Length: 30, Width: 20, Height: 20, Weight: 2}}

// Prices of every courier (or one, via ReturnValuations).
prices, err := c.OrderValuationV2(ctx, polkurier.ValuationRequest{
    OrderRequest: polkurier.OrderRequest{ShipmentType: polkurier.ShipmentBox, Packs: box},
})
for _, p := range prices {
    fmt.Println(p.ServiceCode, p.ServiceName, p.FinalGross().Grosz()) // grosz, promotions applied
}

// Ship to an InPost parcel locker, dropping the parcel off at a locker (no courier pickup).
order, err := c.CreateOrder(ctx, polkurier.OrderRequest{
    ShipmentType: polkurier.ShipmentBox,
    Courier:      polkurier.CourierInPostLocker,
    Description:  "Ceramika",
    Sender:       &polkurier.Address{Person: "Jan Kowalski", Street: "Kurierska", HouseNumber: "1",
                      Postcode: "63-400", City: "Ostrów Wielkopolski", Email: "jan@example.com", Phone: "123456789"},
    Recipient:    &polkurier.Address{Person: "Anna Nowak", Street: "Poznańska", HouseNumber: "12",
                      Postcode: "60-001", City: "Poznań", Email: "anna@example.com", Phone: "987654321", PointID: "POZ01M"},
    Packs:        box,
    Pickup:       &polkurier.Pickup{NoCourierOrder: true},
    Services:     map[polkurier.Service]bool{polkurier.ServiceHandleWithCare: true},
})
label, err := c.GetLabel(ctx, order.OrderNumber) // PDF bytes
status, err := c.GetStatus(ctx, order.OrderNumber)
```

Errors reported by the API (`"status": "error"`) are `*polkurier.APIError` (`polkurier.IsAPIError(err)`);
anything else is a transport problem. Numbers that the API sometimes sends as strings (`"12,30"`) are decoded by
`polkurier.Float`, with `Grosz()` for exact money handling.

## API coverage

| API method | Client method |
|---|---|
| heartbeat | `Heartbeat` |
| test_auth_api | `TestAuth` |
| available_carriers | `AvailableCarriers` |
| order_valuation (deprecated) | `OrderValuation` |
| order_valuation_v2 | `OrderValuationV2` |
| create_order | `CreateOrder` |
| get_label / get_protocol | `GetLabel` / `GetProtocol` |
| get_status | `GetStatus` |
| cancel_order | `CancelOrder` |
| pickup_courier / get_courier_pickup_time | `PickupCourier` / `GetCourierPickupTime` |
| get_countries / get_supported_countries_v2 | `GetCountries` / `GetSupportedCountries` |
| get_courier_point | `GetCourierPoints` |
| get_orders | `GetOrder` / `GetOrders` |
| is_multi_pickup_available / create_multi_order_pickup | `IsMultiPickupAvailable` / `CreateMultiOrderPickup` |
| get_addresses / save_address / delete_address | `GetAddresses` / `SaveAddress` / `DeleteAddress` |
| get_bank_accounts / save_bank_account / delete_bank_account | `GetBankAccounts` / `SaveBankAccount` / `DeleteBankAccount` |
| get_pack_templates / save_pack_template / delete_pack_template | `GetPackTemplates` / `SavePackTemplate` / `DeletePackTemplate` |
| find_city | `FindCity` |
| get_map_token | `GetMapToken` |
| two_factor_token_register / resend / verify | `TwoFactorTokenRegister` / `TwoFactorTokenResend` / `TwoFactorTokenVerify` |

Methods added to the API later can be called with `Client.Call(ctx, "method_name", data, &out)`.

## License

MIT
