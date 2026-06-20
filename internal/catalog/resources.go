// Package catalog maps infrastructure-as-code resource types and container
// images to the concrete data store they back, so Plume can refine a generic
// code-detected store (e.g. "Database") with what the IaC actually declares.
package catalog

// Resource names the generic store node to refine and the concrete label.
type Resource struct {
	EnrichID string
	Label    string
}

// ByToken matches a lowercase token (a Terraform resource type, a compose image
// name, a provider) to a concrete store classification.
var ByToken = map[string]Resource{
	// Terraform / cloud managed databases
	"aws_db_instance":               {"store:db", "Database (Amazon RDS)"},
	"aws_rds_cluster":               {"store:db", "Database (Amazon Aurora)"},
	"google_sql_database_instance":  {"store:db", "Database (Cloud SQL)"},
	"azurerm_postgresql_server":     {"store:db", "PostgreSQL (Azure)"},
	"digitalocean_database_cluster": {"store:db", "Managed database (DigitalOcean)"},

	// engines / images
	"postgres":   {"store:db", "PostgreSQL"},
	"postgresql": {"store:db", "PostgreSQL"},
	"mysql":      {"store:db", "MySQL"},
	"mariadb":    {"store:db", "MariaDB"},
	"cockroach":  {"store:db", "CockroachDB"},

	// object stores
	"aws_s3_bucket":           {"store:object", "Amazon S3 bucket"},
	"google_storage_bucket":   {"store:object", "GCS bucket"},
	"azurerm_storage_account": {"store:object", "Azure Blob storage"},

	// caches
	"aws_elasticache_cluster": {"store:cache", "Redis (ElastiCache)"},
	"redis":                   {"store:cache", "Redis"},
	"memcached":               {"store:cache", "Memcached"},

	// document / nosql
	"aws_dynamodb_table": {"store:dynamo", "Amazon DynamoDB"},
	"mongo":              {"store:mongo", "MongoDB"},
	"mongodb":            {"store:mongo", "MongoDB"},
}
