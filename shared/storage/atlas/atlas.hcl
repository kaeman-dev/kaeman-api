env "pg" {
  src = "file://schema.hcl"
  dev = getenv("ATLAS_DEV_URL")
  url = getenv("KAEMAN_DATABASE_URL")
}
