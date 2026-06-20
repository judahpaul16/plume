package scan

import (
	"regexp"
	"strings"

	"github.com/judahpaul16/plume/internal/graph"
)

// PIICategory is a kind of personal data Plume recognizes by name.
type PIICategory struct {
	ID          string
	Label       string
	Sensitivity graph.Sensitivity
}

// strong maps a single normalized token to a category. These are specific
// enough to classify on their own.
var strong = map[string]PIICategory{
	"email": {"email", "Email", graph.PII}, "emailaddress": {"email", "Email", graph.PII},
	"phone": {"phone", "Phone", graph.PII}, "telephone": {"phone", "Phone", graph.PII},
	"mobile": {"phone", "Phone", graph.PII}, "msisdn": {"phone", "Phone", graph.PII},
	"ssn": {"ssn", "National ID", graph.PII}, "socialsecurity": {"ssn", "National ID", graph.PII},
	"sin": {"ssn", "National ID", graph.PII}, "nino": {"ssn", "National ID", graph.PII},
	"passport": {"govid", "Government ID", graph.PII}, "taxid": {"govid", "Government ID", graph.PII},
	"ein": {"govid", "Government ID", graph.PII}, "vat": {"govid", "Government ID", graph.PII},
	"dob": {"dob", "Date of birth", graph.PII}, "birthdate": {"dob", "Date of birth", graph.PII},
	"dateofbirth": {"dob", "Date of birth", graph.PII}, "birthday": {"dob", "Date of birth", graph.PII},
	"password": {"password", "Password", graph.Credential}, "passwd": {"password", "Password", graph.Credential},
	"pwd": {"password", "Password", graph.Credential}, "secret": {"credential", "Credential", graph.Credential},
	"apikey": {"credential", "Credential", graph.Credential}, "accesstoken": {"credential", "Credential", graph.Credential},
	"refreshtoken": {"credential", "Credential", graph.Credential}, "privatekey": {"credential", "Credential", graph.Credential},
	"clientsecret": {"credential", "Credential", graph.Credential},
	"creditcard":   {"card", "Card details", graph.Financial}, "debitcard": {"card", "Card details", graph.Financial},
	"cardnumber": {"card", "Card details", graph.Financial}, "ccn": {"card", "Card details", graph.Financial},
	"cvv": {"card", "Card details", graph.Financial}, "cvc": {"card", "Card details", graph.Financial},
	"iban": {"bank", "Bank account", graph.Financial}, "accountnumber": {"bank", "Bank account", graph.Financial},
	"routing": {"bank", "Bank account", graph.Financial}, "sortcode": {"bank", "Bank account", graph.Financial},
	"latitude": {"geo", "Location", graph.PII}, "longitude": {"geo", "Location", graph.PII},
	"geolocation": {"geo", "Location", graph.PII}, "gps": {"geo", "Location", graph.PII},
	"ipaddress": {"device", "Device / IP", graph.PII}, "useragent": {"device", "Device / IP", graph.PII},
	"deviceid":  {"device", "Device / IP", graph.PII},
	"diagnosis": {"health", "Health", graph.Health}, "medical": {"health", "Health", graph.Health},
	"patient": {"health", "Health", graph.Health},
	"gender":  {"special", "Special category", graph.Special}, "ethnicity": {"special", "Special category", graph.Special},
	"religion": {"special", "Special category", graph.Special},
	"username": {"username", "Username", graph.PII}, "userid": {"username", "Username", graph.PII},
	"firstname": {"name", "Name", graph.PII}, "lastname": {"name", "Name", graph.PII},
	"fullname": {"name", "Name", graph.PII}, "surname": {"name", "Name", graph.PII},
	"givenname": {"name", "Name", graph.PII}, "familyname": {"name", "Name", graph.PII},
	"streetaddress": {"address", "Address", graph.PII}, "homeaddress": {"address", "Address", graph.PII},
	"billingaddress": {"address", "Address", graph.PII}, "shippingaddress": {"address", "Address", graph.PII},
	"mailingaddress": {"address", "Address", graph.PII}, "postaladdress": {"address", "Address", graph.PII},
}

// weakName tokens classify as a name only next to one of these qualifiers,
// to avoid flagging every "name" variable.
var nameQualifiers = map[string]bool{
	"first": true, "last": true, "full": true, "given": true, "family": true,
	"user": true, "customer": true, "contact": true, "display": true, "legal": true,
}

var addressQualifiers = map[string]bool{
	"street": true, "home": true, "billing": true, "shipping": true,
	"mailing": true, "postal": true, "residential": true, "delivery": true,
}

var camel = regexp.MustCompile(`([a-z0-9])([A-Z])`)

// tokens splits an identifier into lowercase word tokens across camelCase,
// snake_case, kebab-case, and dots.
func tokens(id string) []string {
	s := camel.ReplaceAllString(id, "${1} ${2}")
	s = strings.Map(func(r rune) rune {
		if r == '_' || r == '-' || r == '.' || r == '/' {
			return ' '
		}
		return r
	}, s)
	return strings.Fields(strings.ToLower(s))
}

// Classify decides whether an identifier names personal data.
func Classify(id string) (PIICategory, bool) {
	tk := tokens(id)
	if len(tk) == 0 {
		return PIICategory{}, false
	}
	joined := strings.Join(tk, "")
	if c, ok := strong[joined]; ok {
		return c, true
	}
	for i, t := range tk {
		if c, ok := strong[t]; ok {
			return c, true
		}
		if t == "name" {
			if (i > 0 && nameQualifiers[tk[i-1]]) || (i+1 < len(tk) && nameQualifiers[tk[i+1]]) {
				return PIICategory{"name", "Name", graph.PII}, true
			}
		}
		if t == "address" && ((i > 0 && addressQualifiers[tk[i-1]]) || (i+1 < len(tk) && addressQualifiers[tk[i+1]])) {
			return PIICategory{"address", "Address", graph.PII}, true
		}
	}
	return PIICategory{}, false
}

var placeholder = regexp.MustCompile(`(?i)\b(example\.com|test|foo|bar|baz|john\.?doe|jane\.?doe|lorem|dummy|sample|placeholder|xxx+|changeme|your[-_]?)\b`)

// LooksLikePlaceholder reports whether a literal value is obviously fake, so
// flows derived purely from sample data are dropped.
func LooksLikePlaceholder(v string) bool {
	return placeholder.MatchString(v)
}
