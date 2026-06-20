package scan

import (
	"regexp"
	"sort"
	"strings"

	"github.com/judahpaul16/plume/internal/graph"
)

// callKinds are the AST node types that represent a function/method call across
// the common tree-sitter grammars.
var callKinds = map[string]bool{
	"call_expression": true, "call": true, "function_call": true,
	"method_invocation": true, "method_call": true, "invocation_expression": true,
	"function_call_expression": true, "scoped_call_expression": true,
}

// target is the destination a call routes data to.
type target struct {
	id    string
	label string
	kind  graph.Kind
	via   string
}

var (
	logReceivers = regexp.MustCompile(`(?i)\b(log|logger|logging|console|slog|logrus|zap|winston|pino|klog|glog)\b`)
	logMethods   = regexp.MustCompile(`(?i)^(log|logf|logln|debug|info|warn|warning|error|fatal|trace|print|println|printf|sprintf)$`)

	storeMethods   = regexp.MustCompile(`(?i)^(save|insert|insertone|insertmany|create|upsert|put|putitem|write|persist|store|add|set|setex|hset|exec|execute|query|update|merge|bulkwrite|copy|append|create_all|put_item)$`)
	storeReceivers = regexp.MustCompile(`(?i)\b(db|dba|database|repo|repository|model|models|orm|store|collection|coll|table|tbl|redis|rdb|cache|kv|bucket|s3|gcs|blob|dynamo|dynamodb|mongo|mongodb|prisma|sequelize|knex|gorm|ecto|datastore|session|conn|pool|tx|sql)\b`)

	httpMethods   = regexp.MustCompile(`(?i)^(fetch|request|get|post|put|patch|delete|head|do|send|urlopen|urlretrieve)$`)
	httpReceivers = regexp.MustCompile(`(?i)\b(http|https|axios|requests|urllib|httpx|httpclient|httpoison|finch|got|ky|superagent|undici|net|client|api)\b`)

	commsRe = regexp.MustCompile(`(?i)(sendmail|send_mail|sendemail|send_email|sendsms|send_sms|sendmessage|send_message|mailer|transporter|nodemailer|smtp|postmark|deliver_now|deliver_later|\bnotify\b|publish|enqueue)`)

	urlRe   = regexp.MustCompile(`https?://([a-zA-Z0-9.\-]+)`)
	identRe = regexp.MustCompile(`[A-Za-z_][A-Za-z0-9_]*`)
)

// knownVendors maps a token in a callee to a friendly third-party name.
var knownVendors = map[string]string{
	"stripe": "Stripe", "twilio": "Twilio", "sendgrid": "SendGrid", "mailgun": "Mailgun",
	"segment": "Segment", "mixpanel": "Mixpanel", "amplitude": "Amplitude", "posthog": "PostHog",
	"sentry": "Sentry", "datadog": "Datadog", "slack": "Slack", "openai": "OpenAI",
	"anthropic": "Anthropic", "plaid": "Plaid", "intercom": "Intercom", "hubspot": "HubSpot",
	"mailchimp": "Mailchimp", "braintree": "Braintree", "paypal": "PayPal", "auth0": "Auth0",
}

// vendorTokens is knownVendors' keys in a stable order, so a callee or host that
// contains more than one token resolves the same way every run.
var vendorTokens = func() []string {
	ts := make([]string, 0, len(knownVendors))
	for t := range knownVendors {
		ts = append(ts, t)
	}
	sort.Strings(ts)
	return ts
}()

// matchVendor finds the first known vendor token contained in s (lowercased) and
// returns its id token and friendly name. The same catalog canonicalizes both an
// SDK callee and an HTTP host, so api.anthropic.com folds into the Anthropic node.
func matchVendor(s string) (string, string, bool) {
	for _, tok := range vendorTokens {
		if strings.Contains(s, tok) {
			return tok, knownVendors[tok], true
		}
	}
	return "", "", false
}

func splitCallee(callee string) (recv, method string) {
	callee = strings.TrimSpace(callee)
	if i := strings.LastIndexAny(callee, ".:>"); i >= 0 {
		return callee[:i], strings.Trim(callee[i+1:], ".:>")
	}
	return "", callee
}

// classifyCall decides whether a call routes data to a store, sink, or external,
// given the callee (text before the arguments) and the full call text.
func classifyCall(callee, text string) (target, bool) {
	low := strings.ToLower(callee)
	recv, method := splitCallee(low)

	if tok, name, ok := matchVendor(low); ok {
		return target{"ext:" + tok, name, graph.External, callee}, true
	}
	if commsRe.MatchString(low) {
		return target{"ext:comms", "Email / messaging", graph.External, callee}, true
	}
	if (httpReceivers.MatchString(recv) && httpMethods.MatchString(method)) || method == "fetch" {
		if m := urlRe.FindStringSubmatch(text); m != nil && !LooksLikePlaceholder(m[1]) {
			if tok, name, ok := matchVendor(strings.ToLower(m[1])); ok {
				return target{"ext:" + tok, name, graph.External, callee}, true
			}
			return target{"ext:" + m[1], m[1], graph.External, callee}, true
		}
		return target{"ext:http", "External HTTP", graph.External, callee}, true
	}
	if logReceivers.MatchString(recv) || logMethods.MatchString(method) {
		return target{"sink:log", "Logs", graph.Sink, callee}, true
	}
	if storeMethods.MatchString(method) && (storeReceivers.MatchString(recv) || method == "insert" || method == "upsert" || method == "save") {
		id, label := "store:db", "Data store"
		if r := storeReceiverName(recv); r != "" {
			id, label = "store:"+r, storeLabel(r)
		}
		return target{id, label, graph.Store, callee}, true
	}
	return target{}, false
}

// storeReceiverName normalizes a matched store receiver to a stable key so node
// IDs are consistent and the infra catalog can refine them.
func storeReceiverName(recv string) string {
	switch strings.ToLower(storeReceivers.FindString(recv)) {
	case "redis", "rdb", "cache", "kv":
		return "cache"
	case "s3", "gcs", "blob", "bucket":
		return "object"
	case "mongo", "mongodb":
		return "mongo"
	case "dynamo", "dynamodb":
		return "dynamo"
	case "":
		return ""
	default:
		return "db"
	}
}

func storeLabel(r string) string {
	switch r {
	case "cache":
		return "Redis / cache"
	case "object":
		return "Object store"
	case "mongo":
		return "MongoDB"
	case "dynamo":
		return "DynamoDB"
	default:
		return "Database"
	}
}

// categoriesIn finds the personal-data categories named by identifiers in a
// snippet (typically a call's arguments).
func categoriesIn(snippet string) []PIICategory {
	seen := map[string]bool{}
	var out []PIICategory
	for _, id := range identRe.FindAllString(snippet, -1) {
		if c, ok := Classify(id); ok && !seen[c.ID] {
			seen[c.ID] = true
			out = append(out, c)
		}
	}
	return out
}
