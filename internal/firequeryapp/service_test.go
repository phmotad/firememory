package firequeryapp

import (
	"context"
	"testing"

	"github.com/phmotad/firememory/internal/firequery/models"
)

func TestResolveModelConfigDefaults(t *testing.T) {
	t.Parallel()

	config := ResolveModelConfig(nil)
	if config.IntentModelID != models.IntentModelDeBERTaSmall {
		t.Fatalf("IntentModelID = %q", config.IntentModelID)
	}
	if config.TriggerModelID != models.TriggerModelDeBERTaSmall {
		t.Fatalf("TriggerModelID = %q", config.TriggerModelID)
	}
	if config.EntityModelID != models.EntityModelGLiNER2Small {
		t.Fatalf("EntityModelID = %q", config.EntityModelID)
	}
	if config.SimilarityModelID != models.SimilarityModelE5Small {
		t.Fatalf("SimilarityModelID = %q", config.SimilarityModelID)
	}
}

func TestResolveModelConfigOverrides(t *testing.T) {
	t.Parallel()

	values := map[string]string{
		envIntentModel:     models.IntentModelModernBERT,
		envTriggerModel:    models.TriggerModelModernBERT,
		envEntityModel:     "urchade/gliner_medium-v2.1",
		envSimilarityModel: "intfloat/multilingual-e5-small",
	}

	config := ResolveModelConfig(func(key string) string {
		return values[key]
	})
	if config.IntentModelID != models.IntentModelModernBERT {
		t.Fatalf("IntentModelID = %q", config.IntentModelID)
	}
	if config.TriggerModelID != models.TriggerModelModernBERT {
		t.Fatalf("TriggerModelID = %q", config.TriggerModelID)
	}
	if config.EntityModelID != "urchade/gliner_medium-v2.1" {
		t.Fatalf("EntityModelID = %q", config.EntityModelID)
	}
	if config.SimilarityModelID != "intfloat/multilingual-e5-small" {
		t.Fatalf("SimilarityModelID = %q", config.SimilarityModelID)
	}
}

func TestBuildRuntimeManagerRegistersConfiguredModels(t *testing.T) {
	t.Parallel()

	values := map[string]string{
		envIntentModel:     models.IntentModelModernBERT,
		envTriggerModel:    models.TriggerModelModernBERT,
		envEntityModel:     "urchade/gliner_medium-v2.1",
		envSimilarityModel: "intfloat/multilingual-e5-small",
	}

	manager := BuildRuntimeManager(func(key string) string {
		return values[key]
	})
	health, err := manager.Health(context.Background())
	if err != nil {
		t.Fatalf("Health() error = %v", err)
	}

	ids := map[string]bool{}
	for _, state := range health.Models {
		ids[state.ID] = true
	}
	for _, want := range []string{
		models.IntentModelModernBERT,
		models.TriggerModelModernBERT,
		"urchade/gliner_medium-v2.1",
		"intfloat/multilingual-e5-small",
	} {
		if !ids[want] {
			t.Fatalf("runtime models missing %q in %#v", want, health.Models)
		}
	}
}

func TestRequireRealModelsFalseByDefault(t *testing.T) {
	t.Parallel()

	if envTruthy(nil, envRequireReal) {
		t.Fatal("FIREQUERY_REQUIRE_REAL_MODELS should default to false")
	}
}

func TestRequireRealModelsOverride(t *testing.T) {
	t.Parallel()

	values := map[string]string{envRequireReal: "1"}
	if !envTruthy(func(k string) string { return values[k] }, envRequireReal) {
		t.Fatal("FIREQUERY_REQUIRE_REAL_MODELS=1 should be truthy")
	}
}
