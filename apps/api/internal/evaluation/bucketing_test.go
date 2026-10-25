package evaluation

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"
)

func TestBucketingV1GoldenVectors(t *testing.T) {
	data, err := os.ReadFile("../../../../testdata/bucketing-v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Version int
		Vectors []struct {
			Name, ProjectID, EnvironmentID, FlagKey, Salt, BucketingKey, Digest string
			Bucket                                                              int32
		}
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	if fixture.Version != 1 || len(fixture.Vectors) < 9 {
		t.Fatal("fixture v1 ausente")
	}
	for _, row := range fixture.Vectors {
		t.Run(row.Name, func(t *testing.T) {
			input := BucketInput{row.ProjectID, row.EnvironmentID, row.FlagKey, row.Salt, row.BucketingKey}
			digest := sha256.Sum256([]byte(row.ProjectID + ":" + row.EnvironmentID + ":" + row.FlagKey + ":" + row.Salt + ":" + row.BucketingKey))
			if hex.EncodeToString(digest[:]) != row.Digest || BucketV1(input) != row.Bucket {
				t.Fatal("vetor v1 divergiu")
			}
			for _, bps := range []int32{0, 1, row.Bucket, row.Bucket + 1, 9999, 10000} {
				value, reason, err := Decide(true, true, bps, &input)
				wantReason := "PERCENTAGE_ROLLOUT"
				if bps == 0 {
					wantReason = "DISABLED"
				}
				if bps == 10000 {
					wantReason = "ENABLED"
				}
				if err != nil || value != (row.Bucket < bps) || reason != wantReason {
					t.Fatalf("limiar incorreto em %d", bps)
				}
				value, reason, err = Decide(true, false, bps, &input)
				if err != nil || value || reason != "DISABLED" {
					t.Fatal("desativada recebeu rollout")
				}
			}
			for _, bps := range []int32{1, 5000, 9999} {
				value, reason, err := Decide(true, true, bps, nil)
				if err != nil || value || reason != "MISSING_CONTEXT" {
					t.Fatal("contexto ausente não falhou fechado")
				}
			}
		})
	}
}
