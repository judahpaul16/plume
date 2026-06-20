package scan

import "testing"

func TestClassify(t *testing.T) {
	want := map[string]string{
		"email": "email", "emailAddress": "email", "user_email": "email",
		"firstName": "name", "lastName": "name", "customerName": "name",
		"ssn": "ssn", "cardNumber": "card", "password": "password",
		"phone": "phone", "ipAddress": "device",
	}
	for id, exp := range want {
		c, ok := Classify(id)
		if !ok || c.ID != exp {
			t.Errorf("Classify(%q) = %q,%v; want %q", id, c.ID, ok, exp)
		}
	}
	for _, id := range []string{"name", "count", "index", "total", "handler", "request", "value"} {
		if c, ok := Classify(id); ok {
			t.Errorf("Classify(%q) unexpectedly matched %q", id, c.ID)
		}
	}
}
