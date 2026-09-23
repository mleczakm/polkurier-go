package polkurier_test

import (
	"context"
	"fmt"
	"os"

	polkurier "github.com/mleczakm/polkurier-go"
)

// Price a 2 kg box with every courier, then ship it to an InPost parcel locker.
func Example() {
	ctx := context.Background()
	c := polkurier.New(os.Getenv("POLKURIER_LOGIN"), os.Getenv("POLKURIER_TOKEN"), polkurier.WithSandbox())

	box := []polkurier.Pack{{Length: 30, Width: 20, Height: 20, Weight: 2}}
	prices, err := c.OrderValuationV2(ctx, polkurier.ValuationRequest{
		OrderRequest: polkurier.OrderRequest{ShipmentType: polkurier.ShipmentBox, Packs: box},
	})
	if err != nil {
		fmt.Println(err)
		return
	}
	for _, p := range prices {
		if p.Available {
			fmt.Printf("%s: %.2f zł\n", p.ServiceName, float64(p.FinalGross()))
		}
	}

	order, err := c.CreateOrder(ctx, polkurier.OrderRequest{
		ShipmentType: polkurier.ShipmentBox,
		Courier:      polkurier.CourierInPostLocker,
		Description:  "Ceramika",
		Sender: &polkurier.Address{Person: "Jan Kowalski", Street: "Kurierska", HouseNumber: "1",
			Postcode: "63-400", City: "Ostrów Wielkopolski", Email: "jan@example.com", Phone: "123456789"},
		Recipient: &polkurier.Address{Person: "Anna Nowak", Street: "Poznańska", HouseNumber: "12",
			Postcode: "60-001", City: "Poznań", Email: "anna@example.com", Phone: "987654321", PointID: "POZ01M"},
		Packs:  box,
		Pickup: &polkurier.Pickup{NoCourierOrder: true}, // drop off at a locker
	})
	if err != nil {
		fmt.Println(err)
		return
	}
	label, _ := c.GetLabel(ctx, order.OrderNumber)
	_ = os.WriteFile(order.OrderNumber+".pdf", label, 0o644)
}
