package scan

import "testing"

func TestClassify(t *testing.T) {
	want := map[string]string{
		"email": "email", "emailAddress": "email", "user_email": "email",
		"firstName": "name", "lastName": "name", "customerName": "name",
		"ssn": "ssn", "cardNumber": "card", "creditCard": "card",
		"password": "password", "phone": "phone", "ipAddress": "device",
		"shippingAddress": "address", "billing_address": "address",
	}
	for id, exp := range want {
		c, ok := Classify(id)
		if !ok || c.ID != exp {
			t.Errorf("Classify(%q) = %q,%v; want %q", id, c.ID, ok, exp)
		}
	}
	// Generic technical terms must not classify as personal data.
	for _, id := range []string{
		"name", "count", "index", "total", "handler", "request", "value",
		"card", "soundCard", "card_match", "address", "ipAddr", "coordinates",
	} {
		if c, ok := Classify(id); ok {
			t.Errorf("Classify(%q) unexpectedly matched %q", id, c.ID)
		}
	}
}
