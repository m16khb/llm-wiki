package vault

import "testing"

func TestResolveWithDefaultPrefersExplicitPath(t *testing.T) {
	t.Setenv(EnvVar, "/env/vault")

	got, err := ResolveWithDefault("/explicit/vault", "/context/vault")
	if err != nil {
		t.Fatal(err)
	}
	if got != "/explicit/vault" {
		t.Fatalf("ResolveWithDefault explicit = %q, want /explicit/vault", got)
	}
}

func TestResolveWithDefaultUsesDefaultBeforeEnv(t *testing.T) {
	t.Setenv(EnvVar, "/env/vault")

	got, err := ResolveWithDefault("", "/context/vault")
	if err != nil {
		t.Fatal(err)
	}
	if got != "/context/vault" {
		t.Fatalf("ResolveWithDefault default = %q, want /context/vault", got)
	}
}

func TestResolveWithDefaultFallsBackToEnv(t *testing.T) {
	t.Setenv(EnvVar, "/env/vault")

	got, err := ResolveWithDefault("", "")
	if err != nil {
		t.Fatal(err)
	}
	if got != "/env/vault" {
		t.Fatalf("ResolveWithDefault env = %q, want /env/vault", got)
	}
}
