package provider

import (
	"context"
	"testing"

	tfprovider "github.com/hashicorp/terraform-plugin-framework/provider"
)

func TestProvider_Metadata(t *testing.T) {
	p := New()
	var resp tfprovider.MetadataResponse
	p.Metadata(context.Background(), tfprovider.MetadataRequest{}, &resp)
	if resp.TypeName != "hivehook" {
		t.Errorf("TypeName = %q, want %q", resp.TypeName, "hivehook")
	}
}

func TestProvider_Schema(t *testing.T) {
	p := New()
	var resp tfprovider.SchemaResponse
	p.Schema(context.Background(), tfprovider.SchemaRequest{}, &resp)

	if _, ok := resp.Schema.Attributes["endpoint"]; !ok {
		t.Error("missing endpoint attribute")
	}
	if _, ok := resp.Schema.Attributes["api_key"]; !ok {
		t.Error("missing api_key attribute")
	}
}

func TestProvider_Resources(t *testing.T) {
	p := New().(*hivehookProvider)
	resources := p.Resources(context.Background())

	want := 8
	if len(resources) != want {
		t.Errorf("resource count = %d, want %d", len(resources), want)
	}
}

func TestProvider_DataSources(t *testing.T) {
	p := New().(*hivehookProvider)
	ds := p.DataSources(context.Background())

	want := 4
	if len(ds) != want {
		t.Errorf("data source count = %d, want %d", len(ds), want)
	}
}

func TestNewClient(t *testing.T) {
	c := NewClient("https://app.hivehook.com", "test-key")
	if c.url != "https://app.hivehook.com/graphql" {
		t.Errorf("url = %q, want %q", c.url, "https://app.hivehook.com/graphql")
	}
	if c.apiKey != "test-key" {
		t.Errorf("apiKey = %q, want %q", c.apiKey, "test-key")
	}
}
