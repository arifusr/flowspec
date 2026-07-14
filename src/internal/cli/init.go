package cli

import (
	"fmt"
	"os"
	"path/filepath"
)

// RunInit scaffolds a new FlowSpec project.
func RunInit(dir string) error {
	if dir == "" {
		dir = "."
	}

	dirs := []string{
		"env",
		"requests",
		"flows",
		"shared",
		"data",
		"scripts",
		"specs",
		"reports",
	}

	for _, d := range dirs {
		path := filepath.Join(dir, d)
		if err := os.MkdirAll(path, 0755); err != nil {
			return fmt.Errorf("failed to create %s: %w", path, err)
		}
		fmt.Printf("✓ Created %s/\n", d)
	}

	// apitest.flow
	apitestFlow := `project "My API Tests" {
  version     = "1.0"
  default_env = dev

  env dev     from "env/dev.flow"
  env staging from "env/staging.flow"

  settings {
    timeout    = 30s
    report_dir = "reports/"
  }
}
`
	if err := writeIfNotExists(filepath.Join(dir, "apitest.flow"), apitestFlow); err != nil {
		return err
	}
	fmt.Println("✓ Created apitest.flow")

	// env/dev.flow
	devEnv := `env dev {
  base_url = "http://localhost:8080"
}
`
	if err := writeIfNotExists(filepath.Join(dir, "env", "dev.flow"), devEnv); err != nil {
		return err
	}
	fmt.Println("✓ Created env/dev.flow")

	// env/staging.flow
	stagingEnv := `env staging {
  base_url     = "https://staging-api.example.com"
  access_token = env("STAGING_API_TOKEN")
}
`
	if err := writeIfNotExists(filepath.Join(dir, "env", "staging.flow"), stagingEnv); err != nil {
		return err
	}
	fmt.Println("✓ Created env/staging.flow")

	// shared/auth.flow
	authFlow := `auth BearerAuth {
  header Authorization = "Bearer {{access_token}}"
}
`
	if err := writeIfNotExists(filepath.Join(dir, "shared", "auth.flow"), authFlow); err != nil {
		return err
	}
	fmt.Println("✓ Created shared/auth.flow")

	// .gitignore
	gitignore := `reports/
.env
*.local.flow
`
	if err := writeIfNotExists(filepath.Join(dir, ".gitignore"), gitignore); err != nil {
		return err
	}
	fmt.Println("✓ Created .gitignore")

	fmt.Println("\nProject initialized! Next steps:")
	fmt.Println("  1. Edit env/dev.flow with your API base URL")
	fmt.Println("  2. Create your first request in requests/")
	fmt.Println("  3. Run: apitest run requests/your-request.flow --env dev")
	return nil
}

func writeIfNotExists(path, content string) error {
	if _, err := os.Stat(path); err == nil {
		return nil // file exists, skip
	}
	return os.WriteFile(path, []byte(content), 0644)
}
